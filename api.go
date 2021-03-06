package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/draganm/go-reactor"
	"github.com/netice9/apparatchik/core"
	"github.com/netice9/apparatchik/public"
	"github.com/netice9/apparatchik/ui"
	"github.com/urfave/negroni"

	"github.com/djimenez/iconv-go"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

type ErrorResponse struct {
	Reason string `json:"reason"`
}

type API struct {
	apparatchick *core.Apparatchik
	dockerClient *client.Client
}

type negroniHTTPRouter struct {
	*httprouter.Router
}

func (router *negroniHTTPRouter) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	router.Router.NotFound = next
	router.Router.ServeHTTP(w, r)
}

func healthckeckMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.URL.Path == "/_ping" && r.Method == "GET" {
		rw.Header().Set("Content-Type", "text/plain")
		rw.Write([]byte("OK"))
		return
	}
	next(rw, r)
}

func startHttpServer(apparatchick *core.Apparatchik, dockerClient *client.Client, port int) error {
	api := &API{
		apparatchick: apparatchick,
		dockerClient: dockerClient,
	}
	router := httprouter.New()

	router.PUT("/api/v1.0/applications/:applicationName", api.CreateApplication)
	router.DELETE("/api/v1.0/applications/:applicationName", api.DeleteApplication)

	router.GET("/api/v1.0/applications", api.GetApplications)
	router.GET("/api/v1.0/applications/:applicationName", api.GetApplication)

	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/inspect", api.GetGoalInspect)
	router.GET("/api/v1.0/applications/:applicationName/goals/:goalName/exec", api.ExecSocket)

	reactor := reactor.New(negroni.HandlerFunc(healthckeckMiddleware), NewAuthHandler(), negroni.NewStatic(public.AssetFS()), &negroniHTTPRouter{router})

	err := reactor.AddScreen("/", ui.IndexFactory)
	if err != nil {
		return err
	}

	err = reactor.AddScreen("/add_application", ui.AddApplicationFactory)
	if err != nil {
		return err
	}

	err = reactor.AddScreen("/apps/:application", ui.ApplicationFactory)
	if err != nil {
		return err
	}
	err = reactor.AddScreen("/apps/:application/:goal/xterm", ui.XTermFactory)
	if err != nil {
		return err
	}
	err = reactor.AddScreen("/apps/:application/:goal", ui.GoalFactory)
	if err != nil {
		return err
	}

	bnd := fmt.Sprintf(":%d", port)
	reactor.Serve(bnd)
	return nil

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

	exec, err := a.dockerClient.ContainerExecCreate(context.Background(), *containerID, types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{command},
	})

	if err != nil {
		panic(err)
	}

	hr, err := a.dockerClient.ContainerExecAttach(context.Background(), exec.ID, types.ExecConfig{
		Tty:          true,
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Detach:       false,
		Cmd:          []string{command},
	})

	if err != nil {
		panic(err)
	}

	// stdinPipeReader, stdinPipeWriter := io.Pipe()

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		panic(err)
	}

	go func() {

		go func() {
			defer func() {
				conn.Close()
				hr.CloseWrite()
			}()
			for {
				_, message, err := conn.ReadMessage()

				if err != nil {
					log.Println("WS Copy", err)
					return
				}

				_, err = hr.Conn.Write(message)
				if err != nil {
					log.Println("WS Copy", err)
					return
				}
			}

		}()

	}()

	go func() {
		conn.WriteMessage(websocket.TextMessage, []byte("connected\r\n"))

		wr := WSReaderWriter{conn}
		_, err2 := io.Copy(wr, hr.Reader)
		if err2 != nil {
			log.Error(err)
		}

	}()

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

	if err != nil {
		respondWithError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/api/v1.0/applications/%s", applicationName))
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
