package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/devsisters/cine"
	"github.com/fsouza/go-dockerclient"
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
	applications map[string]*Application
	dockerClient *docker.Client
}

func StartApparatchick(dockerClient *docker.Client) *Apparatchik {
	apparatchick := &Apparatchik{cine.Actor{}, map[string]*Application{}, dockerClient}
	cine.StartActor(apparatchick)
	return apparatchick

}

func (p *Apparatchik) Terminate(errReason error) {
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

func (ap *Apparatchik) NewApplication(name string, config *ApplicationConfiguration) (*Application, error) {

	_, ok := ap.applications[name]

	if ok {
		return nil, errors.New("Application already exists")
	}

	application := NewApplication(name, config, ap.dockerClient)
	ap.applications[name] = application
	return application, nil
}

func (ap *Apparatchik) TerminateApplication(applicationName string) error {

	application, err := ap.ApplicationByName(applicationName)

	if err != nil {
		return err
	}

	// TODO Terminate() executes outside of the lock - figure out concurrency
	application.Terminate()
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
		return nil, errors.New("Application not found")
	}
	return application, nil
}
