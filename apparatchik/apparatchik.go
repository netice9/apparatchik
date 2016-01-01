package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/devsisters/cine"
	"github.com/fsouza/go-dockerclient"
)

var apparatchick = &Apparatchik{}

func main() {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	cine.Init("localhost:8000")

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
	lock         sync.Mutex
}

func (ap *Apparatchik) GetContainerIDForGoal(applicatioName, goalName string) (*string, error) {

	ap.lock.Lock()
	defer ap.lock.Unlock()

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
	ap.lock.Lock()
	defer ap.lock.Unlock()

	_, ok := ap.applications[name]

	if ok {
		return nil, errors.New("Application already exists")
	}

	application := NewApplication(name, config, ap.dockerClient)
	ap.applications[name] = application
	return application, nil
}

func (ap *Apparatchik) Terminate(applicationName string) error {

	application, err := ap.ApplicationByName(applicationName)

	if err != nil {
		return err
	}

	// TODO Terminate() executes outside of the lock - figure out concurrency
	application.Terminate()
	ap.lock.Lock()
	defer ap.lock.Unlock()
	delete(ap.applications, applicationName)
	return nil
}

func (ap *Apparatchik) ApplicatioNames() []string {
	ap.lock.Lock()
	defer ap.lock.Unlock()

	names := []string{}
	for k, _ := range ap.applications {
		names = append(names, k)
	}
	return names
}

func (ap *Apparatchik) ApplicationByName(name string) (*Application, error) {
	ap.lock.Lock()
	defer ap.lock.Unlock()

	application, ok := ap.applications[name]
	if !ok {
		return nil, errors.New("Application not found")
	}
	return application, nil
}
