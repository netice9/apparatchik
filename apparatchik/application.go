package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/cine"
)

type Application struct {
	cine.Actor
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

func (app *Application) GoalStatusUpdate(goalName, status string) {
	cine.Cast(app.Self(), nil, (*Application).goalStatusUpdate, goalName, status)
}

func (app *Application) goalStatusUpdate(goalName, status string) {
	for name, goal := range app.Goals {
		if name != goalName {
			goal.SiblingStatusUpdate(goalName, status)
		}
	}
}

func (app *Application) Status() *ApplicationStatus {
	res, err := cine.Call(app.Self(), (*Application).status)

	if err != nil {
		panic(err)
	}

	status := (*ApplicationStatus)(nil)

	status, _ = res[0].(*ApplicationStatus)

	return status
}

func (app *Application) status() *ApplicationStatus {
	goals := map[string]*GoalStatus{}

	for name, goal := range app.Goals {
		goals[name] = goal.Status()
	}

	return &ApplicationStatus{
		Name:     app.Name,
		Goals:    goals,
		MainGoal: app.MainGoal,
	}
}

func (app *Application) Logs(goalName string, w io.Writer) error {
	return app.Goals[goalName].Logs(w)
}

func (app *Application) Inspect(goalName string) (*docker.Container, error) {
	if app == nil {
		return nil, applicationNotFoundError
	}
	if goal, ok := app.Goals[goalName]; ok {
		return goal.Inspect()
	} else {
		return nil, goalNotFoundError
	}
}

func (app *Application) TransitionLog(goalName string) ([]TransitionLogEntry, error) {
	if app == nil {
		return nil, applicationNotFoundError
	}

	res, err := cine.Call(app.Self(), (*Application).transitionLog, goalName)

	if err != nil {
		panic(err)
	}

	entries := ([]TransitionLogEntry)(nil)

	entries, _ = res[0].([]TransitionLogEntry)

	err2 := (error)(nil)

	err2, _ = res[1].(error)

	return entries, err2
}

func (app *Application) transitionLog(goalName string) ([]TransitionLogEntry, error) {
	if goal, ok := app.Goals[goalName]; ok {
		return goal.TransitionLog(), nil
	} else {
		return nil, goalNotFoundError
	}
}

func (app *Application) Stats(goalName string, since time.Time) (*Stats, error) {

	if app == nil {
		return nil, applicationNotFoundError
	}

	res, err := cine.Call(app.Self(), (*Application).stats, goalName, since)

	if err != nil {
		panic(err)
	}

	stats := (*Stats)(nil)

	stats, _ = res[0].(*Stats)

	err2 := (error)(nil)

	err2, _ = res[1].(error)

	return stats, err2
}

func (app *Application) stats(goalName string, since time.Time) (*Stats, error) {
	if goal, ok := app.Goals[goalName]; ok {
		return goal.Stats(since), nil
	} else {
		return nil, goalNotFoundError
	}
}

func (app *Application) CurrentStats(goalName string) (*docker.Stats, error) {

	if app == nil {
		return nil, applicationNotFoundError
	}

	res, err := cine.Call(app.Self(), (*Application).currentStats, goalName)

	if err != nil {
		panic(err)
	}

	stats := (*docker.Stats)(nil)

	stats, _ = res[0].(*docker.Stats)

	err2 := (error)(nil)

	err2, _ = res[1].(error)

	return stats, err2
}

func (app *Application) currentStats(goalName string) (*docker.Stats, error) {
	if goal, ok := app.Goals[goalName]; ok {
		return goal.CurrentStats(), nil
	} else {
		return nil, goalNotFoundError
	}
}

func (app *Application) startGoals() {

	for goalName := range app.Configuration.Goals {
		app.Goals[goalName] = NewGoal(app, goalName, app.Name, app.Configuration.Goals, app.DockerClient)
	}

	app.Goals[app.MainGoal].Start()
}

func NewApplication(applicationName string, applicationConfiguration *ApplicationConfiguration, dockerClient *docker.Client) *Application {

	fileName := "/applications/" + applicationName + ".json"

	json, err := json.Marshal(applicationConfiguration)

	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(fileName, json, 0644)

	app := &Application{
		Actor:               cine.Actor{},
		Name:                applicationName,
		Configuration:       applicationConfiguration,
		Goals:               map[string]*Goal{},
		MainGoal:            applicationConfiguration.MainGoal,
		ApplicationFileName: fileName,
		DockerClient:        dockerClient,
	}

	cine.StartActor(app)

	cine.Cast(app.Self(), nil, (*Application).startGoals)

	return app

}

func (p *Application) Terminate(errReason error) {
}

func (app *Application) TerminateApplication() {
	_, err := cine.Call(app.Self(), (*Application).terminateApplication)

	if err != nil {
		panic(err)
	}
}

func (app *Application) terminateApplication() {
	os.Remove(app.ApplicationFileName)
	for _, goal := range app.Goals {
		goal.TerminateGoal()
	}
}

func (app *Application) RequestGoalStart(name string) {
	cine.Cast(app.Self(), nil, (*Application).requestGoalStart, name)
}

func (app *Application) requestGoalStart(name string) {
	if goal, ok := app.Goals[name]; ok {
		goal.Start()
	} else {
		log.Error("Application ", app.Name, " requested start of uknown goal ", name)
	}
}

func (app *Application) HandleDockerEvent(evt *docker.APIEvents) {
	cine.Cast(app.Self(), nil, (*Application).handleDockerEvent, evt)
}

func (app *Application) handleDockerEvent(evt *docker.APIEvents) {
	for _, goal := range app.Goals {
		goal.HandleDockerEvent(evt)
	}
}
