package core

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/draganm/emission"
	"github.com/fsouza/go-dockerclient"
)

const MaxListeners = 500

var (
	ErrApplicationAlreadyExists = errors.New("Application already exists")
	ErrApplicationNotFound      = errors.New("Application not found")
	ErrGoalNotFound             = errors.New("Goal not found")
)

type Application struct {
	sync.Mutex
	Name                string
	Configuration       *ApplicationConfiguration
	Goals               map[string]*Goal
	MainGoal            string
	ApplicationFileName string
	DockerClient        *docker.Client
	*emission.Emitter
}

type ApplicationStatus struct {
	Name     string                `json:"name"`
	Goals    map[string]GoalStatus `json:"goals"`
	MainGoal string                `json:"main_goal"`
}

func NewApplicationStatus() ApplicationStatus {
	return ApplicationStatus{
		Goals: map[string]GoalStatus{},
	}
}

func (a *Application) GoalStatusUpdate(goalName, status string) {

	for name, goal := range a.Goals {
		if name != goalName {
			goal.SiblingStatusUpdate(goalName, status)
		}
	}
	a.Emitter.Emit("update", a.Status())
}

func (a *Application) Status() ApplicationStatus {

	goalStats := map[string]GoalStatus{}

	for name, goal := range a.Goals {
		goalStats[name] = goal.Status()
	}

	return ApplicationStatus{
		Name:     a.Name,
		Goals:    goalStats,
		MainGoal: a.MainGoal,
	}
}

func (a *Application) goalByName(goalName string) (*Goal, error) {
	a.Lock()
	defer a.Unlock()

	goal, found := a.Goals[goalName]
	if !found {
		return nil, ErrGoalNotFound
	}
	return goal, nil
}
func (a *Application) Logs(goalName string, w io.Writer) error {
	goal, err := a.goalByName(goalName)
	if err != nil {
		return err
	}

	return goal.Logs(w)
}

func (a *Application) Inspect(goalName string) (*docker.Container, error) {
	if a == nil {
		return nil, ErrApplicationNotFound
	}
	goal, err := a.goalByName(goalName)
	if err != nil {
		return nil, err
	}
	return goal.Inspect()
}

func (a *Application) TransitionLog(goalName string) ([]TransitionLogEntry, error) {
	goal, err := a.goalByName(goalName)
	if err != nil {
		return nil, err
	}
	return goal.GetTransitionLog(), nil
}

func (a *Application) Stats(goalName string, since time.Time) (*Stats, error) {
	goal, err := a.goalByName(goalName)
	if err != nil {
		return nil, err
	}
	return goal.Stats(since), nil
}

func (a *Application) CurrentStats(goalName string) (*docker.Stats, error) {
	goal, err := a.goalByName(goalName)
	if err != nil {
		return nil, err
	}
	return goal.CurrentStats(), nil
}

func (a *Application) startGoals() {
	a.Lock()
	for goalName := range a.Configuration.Goals {

		a.Goals[goalName] = NewGoal(a, goalName, a.Name, a.Configuration.Goals, a.DockerClient)
	}
	a.Unlock()
	a.Goals[a.MainGoal].Start()
}

func NewApplicationWithDockerClientFromEnv(applicationName string, applicationConfiguration *ApplicationConfiguration) (*Application, error) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	dockerEventsChannel := make(chan *docker.APIEvents, 20)
	err = dockerClient.AddEventListener(dockerEventsChannel)
	if err != nil {
		return nil, err
	}

	application := NewApplication(applicationName, applicationConfiguration, dockerClient)
	go func() {
		for evt := range dockerEventsChannel {
			application.HandleDockerEvent(evt)
		}
	}()
	return application, nil
}

func NewApplication(applicationName string, applicationConfiguration *ApplicationConfiguration, dockerClient *docker.Client) *Application {

	fileName := "/applications/" + applicationName + ".json"

	json, err := json.Marshal(applicationConfiguration)

	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(fileName, json, 0644)

	emitter := emission.NewEmitter()
	emitter.SetMaxListeners(MaxListeners)

	app := &Application{
		Name:                applicationName,
		Configuration:       applicationConfiguration,
		Goals:               map[string]*Goal{},
		MainGoal:            applicationConfiguration.MainGoal,
		ApplicationFileName: fileName,
		DockerClient:        dockerClient,
		Emitter:             emitter,
	}

	app.startGoals()

	app.Emit("update", app.Status())

	return app

}

func (a *Application) TerminateApplication() {
	os.Remove(a.ApplicationFileName)
	for _, goal := range a.Goals {
		goal.TerminateGoal()
	}

	a.Emit("terminated")
}

func (a *Application) RequestGoalStart(name string) {

	if goal, ok := a.Goals[name]; ok {
		goal.Start()
		return
	}
	log.Warn("Application ", a.Name, " requested start of uknown goal ", name)

}

func (a *Application) HandleDockerEvent(evt *docker.APIEvents) {

	for _, goal := range a.Goals {
		goal.HandleDockerEvent(evt)
	}
}
