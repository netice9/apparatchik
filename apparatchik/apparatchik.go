package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/cine"
)

var apparatchick *Apparatchik = nil

func main() {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	cine.Init("localhost:8000")

	apparatchick = StartApparatchick(dockerClient)

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
	cine.Actor
	applications        map[string]*Application
	dockerClient        *docker.Client
	dockerEventsChannel chan *docker.APIEvents
}

func StartApparatchick(dockerClient *docker.Client) *Apparatchik {

	dockerEventsChannel := make(chan *docker.APIEvents, 20)
	err := dockerClient.AddEventListener(dockerEventsChannel)
	if err != nil {
		panic(err)
	}

	apparatchick := &Apparatchik{cine.Actor{}, map[string]*Application{}, dockerClient, dockerEventsChannel}
	cine.StartActor(apparatchick)

	go func() {
		for evt := range dockerEventsChannel {
			apparatchick.HandleDockerEvent(evt)
		}
	}()

	return apparatchick

}

func (p *Apparatchik) Terminate(errReason error) {
	for _, application := range p.applications {
		application.TerminateApplication()
	}
	p.applications = map[string]*Application{}
}

func (p *Apparatchik) HandleDockerEvent(evt *docker.APIEvents) {
	cine.Cast(p.Self(), nil, (*Apparatchik).handleDockerEvent, evt)
}

func (p *Apparatchik) handleDockerEvent(evt *docker.APIEvents) {
	for _, application := range p.applications {
		application.HandleDockerEvent(evt)
	}
}

func (p *Apparatchik) applicationStatus(applicatioName string) (*ApplicationStatus, error) {
	app, err := p.ApplicationByName(applicatioName)
	if err != nil {
		return nil, err
	}
	return app.Status(), nil
}

func (p *Apparatchik) ApplicationStatus(applicatioName string) (*ApplicationStatus, error) {
	res, err := cine.Call(apparatchick.Self(), (*Apparatchik).applicationStatus, applicatioName)

	if err != nil {
		panic(err)
	}

	status := (*ApplicationStatus)(nil)

	status, _ = res[0].(*ApplicationStatus)

	err2 := (error)(nil)

	err2, _ = res[1].(error)

	return status, err2
}

func (ap *Apparatchik) GetContainerIDForGoal(applicatioName, goalName string) (*string, error) {

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

func (ap *Apparatchik) NewApplication(name string, config *ApplicationConfiguration) (*ApplicationStatus, error) {
	res, err := cine.Call(apparatchick.Self(), (*Apparatchik).newApplication, name, config)

	if err != nil {
		panic(err)
	}

	status := (*ApplicationStatus)(nil)

	status, _ = res[0].(*ApplicationStatus)

	err2 := (error)(nil)

	err2, _ = res[1].(error)

	return status, err2
}

func (ap *Apparatchik) newApplication(name string, config *ApplicationConfiguration) (*ApplicationStatus, error) {

	_, ok := ap.applications[name]

	if ok {
		return nil, applicationAlreadyExistsError
	}

	application := NewApplication(name, config, ap.dockerClient)
	ap.applications[name] = application
	return application.Status(), nil
}

func (ap *Apparatchik) TerminateApplication(applicationName string) error {

	res, err := cine.Call(apparatchick.Self(), (*Apparatchik).terminateApplication, applicationName)

	if err != nil {
		panic(err)
	}

	err2 := (error)(nil)

	err2, _ = res[0].(error)

	return err2
}

func (ap *Apparatchik) terminateApplication(applicationName string) error {

	application, err := ap.ApplicationByName(applicationName)

	if err != nil {
		return err
	}

	application.TerminateApplication()
	delete(ap.applications, applicationName)
	return nil
}

func (ap *Apparatchik) ApplicatioNames() []string {

	res, err := cine.Call(apparatchick.Self(), (*Apparatchik).applicatioNames)

	if err != nil {
		panic(err)
	}

	return res[0].([]string)
}

func (ap *Apparatchik) applicatioNames() []string {

	names := []string{}
	for k, _ := range ap.applications {
		names = append(names, k)
	}
	return names
}

func (ap *Apparatchik) ApplicationByName(name string) (*Application, error) {
	application, ok := ap.applications[name]
	if !ok {
		return nil, applicationNotFoundError
	}
	return application, nil
}

func (ap *Apparatchik) GoalTransitionLog(applicationName, goalName string) ([]TransitionLogEntry, error) {
	res, err := cine.Call(apparatchick.Self(), (*Apparatchik).goalTransitionLog, applicationName, goalName)

	if err != nil {
		panic(err)
	}

	logEntries := ([]TransitionLogEntry)(nil)

	logEntries, _ = res[0].([]TransitionLogEntry)

	err2 := (error)(nil)

	err2, _ = res[1].(error)

	return logEntries, err2
}

func (ap *Apparatchik) goalTransitionLog(applicationName, goalName string) ([]TransitionLogEntry, error) {
	application, err := ap.ApplicationByName(applicationName)
	if err != nil {
		return nil, err
	}

	return application.TransitionLog(goalName)
}
