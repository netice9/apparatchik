package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/djimenez/iconv-go"
	"github.com/fsouza/go-dockerclient"
	"github.com/gorilla/context"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

type ErrorResponse struct {
	Reason string `json:"reason"`
}

var (
	applicationAlreadyExistsError = errors.New("Application already exists")
	applicationNotFoundError      = errors.New("Application not found")
	goalNotFoundError             = errors.New("Goal not found")
)

func startHttpServer() {
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

func GetApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	status, err := apparatchick.ApplicationStatus(applicationName)

	w.Header().Set("Content-Type", "application/json")
	if err == nil {
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(status); err != nil {
			panic(err)
		}
	} else {
		respondWithError(err, w)
		return
	}
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

func DeleteApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")

	err := apparatchick.TerminateApplication(applicationName)

	if err != nil {
		respondWithError(err, w)
		return
	}

	w.WriteHeader(204)

}

func GetApplications(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(apparatchick.ApplicatioNames()); err != nil {
		panic(err)
	}

}

func GetGoalLogs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")
	application, err := apparatchick.ApplicationByName(applicationName)
	if err != nil {
		respondWithError(err, w)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	application.Logs(goalName, w)
}

func GetGoalTransitionLog(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	transitionLog, err := apparatchick.GoalTransitionLog(applicationName, goalName)

	if err != nil {
		respondWithError(err, w)
		return
	}

	if transitionLog != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(transitionLog); err != nil {
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

	application, err := apparatchick.ApplicationByName(applicationName)

	if err != nil {
		respondWithError(err, w)
		return
	}

	stats, err := application.Stats(goalName, sinceTime)
	if err != nil {
		respondWithError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		panic(err)
	}
}

func GetGoalCurrentStats(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	application, err := apparatchick.ApplicationByName(applicationName)

	if err != nil {
		respondWithError(err, w)
		return
	}

	stats, err := application.CurrentStats(goalName)
	if err != nil {
		respondWithError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		panic(err)
	}

}

func GetGoalInspect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	application, err := apparatchick.ApplicationByName(applicationName)

	if err != nil {
		respondWithError(err, w)
		return
	}

	container, err := application.Inspect(goalName)

	if err != nil {
		respondWithError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(container); err != nil {
		panic(err)
	}

}

func CreateApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	decoder := json.NewDecoder(r.Body)

	var applicationConfiguration ApplicationConfiguration
	err := decoder.Decode(&applicationConfiguration)

	if err != nil {
		respondWithError(err, w)
		return
	}

	err = applicationConfiguration.Validate()

	if err != nil {
		respondWithError(err, w)
		return
	}

	status, err := apparatchick.NewApplication(applicationName, &applicationConfiguration)

	log.Info("application created and status returned")

	if err != nil {
		respondWithError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		panic(err)
	}

}

func respondWithError(err error, w http.ResponseWriter) {
	code := 500
	if err == applicationNotFoundError || err == goalNotFoundError {
		code = 404
	} else if err == applicationAlreadyExistsError {
		code = 409
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	e := ErrorResponse{Reason: err.Error()}
	if err := json.NewEncoder(w).Encode(e); err != nil {
		panic(err)
	}
	return
}
