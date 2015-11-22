package main

import (
	"encoding/json"
	"github.com/fsouza/go-dockerclient"
	"io"
	"io/ioutil"
	"os"
	"time"
)

type ApplicationConfiguration struct {
	Goals    map[string]GoalConfiguration `json:"goals"`
	MainGoal string                       `json:"main_goal"`
}

type Application struct {
	Name                string
	Goals               map[string]*Goal
	MainGoal            string
	ApplicationFileName string

	StatusRequest    chan chan *ApplicationStatus
	TerminateRequest chan bool
}

type ApplicationStatus struct {
	Name     string                 `json:"name"`
	Goals    map[string]*GoalStatus `json:"goals"`
	MainGoal string                 `json:"main_goal"`
}

func (app *Application) Actor() {

	app.startGoals()

	for {
		select {
		case <-app.TerminateRequest:
			os.Remove(app.ApplicationFileName)
			for _, goal := range app.Goals {
				goal.Terminate()
			}
			return

		case responseChannel := <-app.StatusRequest:

			go func() {
				goals := make(map[string]*GoalStatus)

				for name, goal := range app.Goals {
					goals[name] = goal.Status()
				}

				responseChannel <- &ApplicationStatus{
					Name:     app.Name,
					Goals:    goals,
					MainGoal: app.MainGoal,
				}
			}()

		}
	}

}

func (app *Application) Status() *ApplicationStatus {

	responseChannel := make(chan *ApplicationStatus)

	app.StatusRequest <- responseChannel

	return <-responseChannel

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

func NewApplication(applicationName string, applicationConfiguration ApplicationConfiguration, dockerClient *docker.Client) *Application {

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

	app := &Application{
		Name:                applicationName,
		StatusRequest:       make(chan chan *ApplicationStatus),
		TerminateRequest:    make(chan bool),
		Goals:               goals,
		MainGoal:            applicationConfiguration.MainGoal,
		ApplicationFileName: fileName,
	}

	go app.Actor()
	return app

}

func (app *Application) Terminate() {
	app.TerminateRequest <- true
}
