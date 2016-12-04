package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/netice9/apparatchik/apparatchik/core"
	"github.com/netice9/apparatchik/apparatchik/ui"
	"github.com/urfave/negroni"
	"gitlab.netice9.com/dragan/go-reactor"

	"github.com/djimenez/iconv-go"
	"github.com/fsouza/go-dockerclient"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

type ErrorResponse struct {
	Reason string `json:"reason"`
}

type API struct {
	apparatchick *core.Apparatchik
	dockerClient *docker.Client
}

type negroniHTTPRouter struct {
	*httprouter.Router
}

func (router *negroniHTTPRouter) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	router.Router.NotFound = next
	router.Router.ServeHTTP(w, r)
}

func startHttpServer(apparatchick *core.Apparatchik, dockerClient *docker.Client) {
	api := &API{
		apparatchick: apparatchick,
		dockerClient: dockerClient,
	}
	router := httprouter.New()

	router.PUT("/api/v1.0/applications/:applicationName", api.CreateApplication)
	router.DELETE("/api/v1.0/applications/:applicationName", api.DeleteApplication)

	router.GET("/api/v1.0/applications", api.GetApplications)
	router.GET("/api/v1.0/applications/:applicationName", api.GetApplication)

	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/logs", api.GetGoalLogs)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/transition_log", api.GetGoalTransitionLog)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/stats", api.GetGoalStats)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/current_stats", api.GetGoalCurrentStats)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/inspect", api.GetGoalInspect)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/exec", api.ExecSocket)

	reactor := reactor.New(NewAuthHandler(), negroni.NewStatic(http.Dir("public")), &negroniHTTPRouter{router})

	bnd := ":8080"

	port := os.Getenv("PORT")

	if port != "" {
		bnd = ":" + port
	}

	reactor.AddScreen("/", ui.IndexFactory)
	reactor.AddScreen("/add_application", ui.AddApplicationFactory)
	reactor.AddScreen("/apps/:application", ui.ApplicationFactory)
	reactor.Serve(bnd)

}

func (a *API) GetApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	status, err := a.apparatchick.ApplicationStatus(applicationName)

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

func (a *API) ExecSocket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	containerID, err := a.apparatchick.GetContainerIDForGoal(applicationName, goalName)

	if err != nil {
		log.Panic(err)
	}

	command := r.FormValue("command")

	if command == "" {
		command = "/bin/sh"
	}

	exec, err := a.dockerClient.CreateExec(docker.CreateExecOptions{
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
	a.dockerClient.StartExec(exec.ID, docker.StartExecOptions{
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

func (a *API) RedirectToIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	http.Redirect(w, r, "/index.html", 301)
}

func (a *API) DeleteApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")

	err := a.apparatchick.TerminateApplication(applicationName)

	if err != nil {
		respondWithError(err, w)
		return
	}

	w.WriteHeader(204)

}

func (a *API) GetApplications(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(a.apparatchick.ApplicatioNames()); err != nil {
		panic(err)
	}

}

func (a *API) GetGoalLogs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")
	application, err := a.apparatchick.ApplicationByName(applicationName)
	if err != nil {
		respondWithError(err, w)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	application.Logs(goalName, w)
}

func (a *API) GetGoalTransitionLog(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	transitionLog, err := a.apparatchick.GoalTransitionLog(applicationName, goalName)

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

func (a *API) GetGoalStats(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")
	sinceString := r.FormValue("since")

	sinceTime, err := time.Parse(time.RFC3339Nano, sinceString)

	if err != nil {
		sinceTime = time.Time{}
	}

	application, err := a.apparatchick.ApplicationByName(applicationName)

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

func (a *API) GetGoalCurrentStats(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	application, err := a.apparatchick.ApplicationByName(applicationName)

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

func (a *API) GetGoalInspect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	goalName := ps.ByName("goalName")

	application, err := a.apparatchick.ApplicationByName(applicationName)

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

func (a *API) CreateApplication(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	applicationName := ps.ByName("applicationName")
	decoder := json.NewDecoder(r.Body)

	var applicationConfiguration core.ApplicationConfiguration
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

	status, err := a.apparatchick.NewApplication(applicationName, &applicationConfiguration)

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
	if err == core.ErrApplicationNotFound || err == core.ErrGoalNotFound {
		code = 404
	} else if err == core.ErrApplicationAlreadyExists {
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
