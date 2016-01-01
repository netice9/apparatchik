package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/devsisters/cine"
	"github.com/fsouza/go-dockerclient"
)

type Application struct {
	cine.Actor
	Name                string
	Goals               map[string]*Goal
	MainGoal            string
	ApplicationFileName string
}

type ApplicationStatus struct {
	Name     string                 `json:"name"`
	Goals    map[string]*GoalStatus `json:"goals"`
	MainGoal string                 `json:"main_goal"`
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
		return nil, nil
	}
	if goal, ok := app.Goals[goalName]; ok {
		return goal.Inspect()
	} else {
		return nil, nil
	}
}

func (app *Application) TransitionLog(goalName string) ([]TransitionLogEntry, error) {
	if app == nil {
		return nil, nil
	}
	if goal, ok := app.Goals[goalName]; ok {
		return goal.TransitionLog(), nil
	} else {
		return nil, nil
	}
}

func (app *Application) Stats(goalName string, since time.Time) (*Stats, error) {
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
	if app == nil {
		return nil, nil
	}
	if goal, ok := app.Goals[goalName]; ok {
		return goal.Stats(since), nil
	} else {
		return nil, nil
	}
}

func (app *Application) CurrentStats(goalName string) (*docker.Stats, error) {
	if app == nil {
		return nil, nil
	}
	if goal, ok := app.Goals[goalName]; ok {
		return goal.CurrentStats(), nil
	} else {
		return nil, nil
	}
}

func (app *Application) startGoals() {
	for _, goal := range app.Goals {
		goal.ApplicationStarted <- app.Goals
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

	goals := make(map[string]*Goal)

	for goalName, _ := range applicationConfiguration.Goals {
		goals[goalName] = NewGoal(goalName, applicationName, applicationConfiguration.Goals, dockerClient)
	}

	app := &Application{cine.Actor{}, applicationName, goals, applicationConfiguration.MainGoal, fileName}

	cine.StartActor(app)

	// TODO: cast the actor
	app.startGoals()

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
		goal.Terminate()
	}
}
