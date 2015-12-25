package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/djimenez/iconv-go"
	"github.com/fsouza/go-dockerclient"
	"github.com/gorilla/context"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

// var applications = Applications{}
// var dockerClient *docker.Client = nil
//

var apparatchick = &Apparatchik{}

func main() {
	endpoint := "unix:///var/run/docker.sock"
	dockerClient, err := docker.NewClient(endpoint)
	// dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	apparatchick.applications = map[string]*Application{}
	apparatchick.dockerClient = dockerClient

	files, err := ioutil.ReadDir("/applications")

	if err != nil {
		panic(err)
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".json") {
			applicationName := name[0 : len(name)-len(".json")]
			data, err := ioutil.ReadFile("/applications/" + name)
			if err != nil {
				panic(err)
			}

			config := ApplicationConfiguration{}

			if err = json.Unmarshal(data, &config); err != nil {
				panic(err)
			}

			apparatchick.NewApplication(applicationName, &config)

		}

	}

	startHttpServer()
}

type Apparatchik struct {
	applications map[string]*Application
	dockerClient *docker.Client
}

func (ap *Apparatchik) GetContainerIDForGoal(applicatioName, goalName string) (*string, error) {

	// TODO locking

	application, ok := ap.applications[applicatioName]
	if !ok {
		return nil, errors.New("Application not found")
	}
	goal, ok := application.Goals[goalName]
	if !ok {
		return nil, errors.New("Goal not found")
	}
	return goal.ContainerId, nil
}

func (ap *Apparatchik) NewApplication(name string, config *ApplicationConfiguration) (*Application, error) {

	// TODO handle application name collision, locking

	application := NewApplication(name, config, ap.dockerClient)
	ap.applications[name] = application
	return application, nil
}

type ErrorResponse struct {
	Reason string `json:"reason"`
}

func startHttpServer() {

	// applications := make(Applications)

	router := httprouter.New()
	router.PUT("/api/v1.0/applications/:applicationName", CreateApplication)
	router.DELETE("/api/v1.0/applications/:applicationName", DeleteApplication)

	router.GET("/api/v1.0/applications", GetApplications)
	router.GET("/api/v1.0/applications/:applicationName", GetApplication)

	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/logs", GetGoalLogs)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/transition_log", GetGoalTransitionLog)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/stats", GetGoalStats)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/current_stats", GetGoalCurrentStats)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/inspect", GetGoalInspect)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/exec", ExecSocket)
	router.NotFound = http.FileServer(http.Dir("public"))

	handler := context.ClearHandler(NewAuthHandler(router))
	http.ListenAndServe(":8080", handler)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WSReaderWriter struct {
	*websocket.Conn
}

func (conn WSReaderWriter) Write(p []byte) (n int, err error) {

	output, err := iconv.ConvertString(string(p), "ISO-8859-1", "utf-8")

	if err != nil {
		log.Println("WSReaderWriter", err)
		return 0, err
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(output))

	if err != nil {
		log.Println("WSReaderWriter", err)
	}

	return len(p), err
}

func ExecSocket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	containerID, err := apparatchick.GetContainerIDForGoal(applicationName, goalName)

	if err != nil {
		log.Panic(err)
	}

	command := r.FormValue("command")

	if command == "" {
		command = "/bin/sh"
	}

	exec, err := apparatchick.dockerClient.CreateExec(docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{command},
		Container:    *containerID,
	})

	if err != nil {
		panic(err)
	}

	stdinPipeReader, stdinPipeWriter := io.Pipe()

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		panic(err)
	}

	go func() {

		conn.WriteMessage(websocket.TextMessage, []byte("connected\r\n"))

		go func() {
			defer func() {
				conn.Close()
				stdinPipeWriter.Close()
			}()
			for {
				_, message, err := conn.ReadMessage()

				if err != nil {
					log.Println("WS Copy", err)
					return
				}
				_, err = stdinPipeWriter.Write(message)
				if err != nil {
					log.Println("WS Copy", err)
					return
				}
			}

		}()

	}()

	// TODO encapsulate into Apparatchik
	apparatchick.dockerClient.StartExec(exec.ID, docker.StartExecOptions{
		Detach:       false,
		Tty:          true,
		InputStream:  stdinPipeReader,
		OutputStream: WSReaderWriter{conn},
		ErrorStream:  WSReaderWriter{conn},
		RawTerminal:  true,
	})

	if err != nil {
		panic(err)
	}

}

func RedirectToIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	http.Redirect(w, r, "/index.html", 301)
}

func (ap *Apparatchik) Terminate(applicationName string) error {
	application, ok := ap.applications[applicationName]
	if !ok {
		return errors.New("Could not find Application")
	}
	application.Terminate()
	delete(ap.applications, applicationName)
	return nil
}

func DeleteApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	// TODO handle error case
	apparatchick.Terminate(applicationName)
	w.WriteHeader(204)
}

func (ap *Apparatchik) ApplicatioNames() []string {
	names := []string{}
	for k, _ := range ap.applications {
		names = append(names, k)
	}
	return names
}

func GetApplications(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apparatchick.ApplicatioNames()); err != nil {
		panic(err)
	}
}

func (ap *Apparatchik) ApplicationByName(name string) (*Application, error) {
	// TODO locking
	application, ok := ap.applications[name]
	if !ok {
		return nil, errors.New("Application not found")
	}
	return application, nil
}

func GetApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	application, err := apparatchick.ApplicationByName(applicationName)

	w.Header().Set("Content-Type", "application/json")
	if err == nil {
		status := application.Status()
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(status); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(404)
		e := ErrorResponse{Reason: "application not found"}
		if err := json.NewEncoder(w).Encode(e); err != nil {
			panic(err)
		}
	}
}

func GetGoalLogs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	// TODO handle error
	application, _ := apparatchick.ApplicationByName(applicationName)
	application.Logs(goalName, w)
}

func GetGoalTransitionLog(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")
	// TODO handle error
	application, _ := apparatchick.ApplicationByName(applicationName)
	transitionLog, err := application.TransitionLog(goalName)
	if err != nil {
		panic(err)
	}
	if transitionLog != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(transitionLog); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(404)
		e := ErrorResponse{Reason: "goal not found"}
		if err := json.NewEncoder(w).Encode(e); err != nil {
			panic(err)
		}
	}
}

func GetGoalStats(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")
	sinceString := r.FormValue("since")

	sinceTime, err := time.Parse(time.RFC3339Nano, sinceString)

	if err != nil {
		sinceTime = time.Time{}
	}

	// TODO handle error
	application, _ := apparatchick.ApplicationByName(applicationName)

	stats, err := application.Stats(goalName, sinceTime)
	if err != nil {
		panic(err)
	}
	if stats != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(404)
		e := ErrorResponse{Reason: "goal not found"}
		if err := json.NewEncoder(w).Encode(e); err != nil {
			panic(err)
		}
	}
}

func GetGoalCurrentStats(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")
	// TODO handle error
	application, _ := apparatchick.ApplicationByName(applicationName)

	stats, err := application.CurrentStats(goalName)
	if err != nil {
		panic(err)
	}
	if stats != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(404)
		e := ErrorResponse{Reason: "goal not found"}
		if err := json.NewEncoder(w).Encode(e); err != nil {
			panic(err)
		}
	}

}

func GetGoalInspect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	// TODO handle error
	application, _ := apparatchick.ApplicationByName(applicationName)

	container, err := application.Inspect(goalName)
	if err != nil {
		panic(err)
	}
	if container != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(container); err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(404)
		e := ErrorResponse{Reason: "goal not found"}
		if err := json.NewEncoder(w).Encode(e); err != nil {
			panic(err)
		}
	}

}

func CreateApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	decoder := json.NewDecoder(r.Body)

	// TODO handle error
	_, err := apparatchick.ApplicationByName(applicationName)

	if err == nil {
		w.WriteHeader(409)
		e := ErrorResponse{Reason: "application already exists"}
		if err := json.NewEncoder(w).Encode(e); err != nil {
			panic(err)
		}
	} else {

		var applicationConfiguration ApplicationConfiguration
		err := decoder.Decode(&applicationConfiguration)

		if err != nil {
			panic("can't parse")
		}

		log.Println("about to create", applicationName, applicationConfiguration)
		// TODO handle error to simplify above if
		app, _ := apparatchick.NewApplication(applicationName, &applicationConfiguration)

		log.Println("created")

		status := app.Status()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)

		if err := json.NewEncoder(w).Encode(status); err != nil {
			panic(err)
		}
	}

}
