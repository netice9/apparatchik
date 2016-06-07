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
	"github.com/fsouza/go-dockerclient"
)

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
}

type ApplicationStatus struct {
	Name     string                 `json:"name"`
	Goals    map[string]*GoalStatus `json:"goals"`
	MainGoal string                 `json:"main_goal"`
}

func (a *Application) GoalStatusUpdate(goalName, status string) {
	goals := a.copyGoals()
	for name, goal := range goals {
		if name != goalName {
			goal.SiblingStatusUpdate(goalName, status)
		}
	}
}

func (a *Application) copyGoals() map[string]*Goal {
	a.Lock()
	defer a.Unlock()
	goals := map[string]*Goal{}

	for name, goal := range a.Goals {
		goals[name] = goal
	}

	return goals
}

func (a *Application) Status() *ApplicationStatus {
	goals := a.copyGoals()
	goalStats := map[string]*GoalStatus{}

	for name, goal := range goals {
		goalStats[name] = goal.Status()
	}

	return &ApplicationStatus{
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
	return NewApplication(applicationName, applicationConfiguration, dockerClient), nil
}

func NewApplication(applicationName string, applicationConfiguration *ApplicationConfiguration, dockerClient *docker.Client) *Application {

	fileName := "/applications/" + applicationName + ".json"

	json, err := json.Marshal(applicationConfiguration)

	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(fileName, json, 0644)

	app := &Application{
		Name:                applicationName,
		Configuration:       applicationConfiguration,
		Goals:               map[string]*Goal{},
		MainGoal:            applicationConfiguration.MainGoal,
		ApplicationFileName: fileName,
		DockerClient:        dockerClient,
	}

	app.startGoals()

	return app

}

func (a *Application) TerminateApplication() {
	os.Remove(a.ApplicationFileName)
	for _, goal := range a.Goals {
		goal.TerminateGoal()
	}
}

func (a *Application) RequestGoalStart(name string) {

	goals := a.copyGoals()

	if goal, ok := goals[name]; ok {
		goal.Start()
		return
	}
	log.Warn("Application ", a.Name, " requested start of uknown goal ", name)

}

func (a *Application) HandleDockerEvent(evt *docker.APIEvents) {
	goals := a.copyGoals()

	for _, goal := range goals {
		goal.HandleDockerEvent(evt)
	}
}
