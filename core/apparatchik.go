package core

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/draganm/emission"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

var ApparatchikInstance *Apparatchik

type Apparatchik struct {
	sync.Mutex
	applications map[string]*Application
	dockerClient *client.Client
	*emission.Emitter
}

func StartApparatchik(dockerClient *client.Client) (*Apparatchik, error) {

	ch, _ := dockerClient.Events(context.Background(), types.EventsOptions{})

	apparatchick := &Apparatchik{
		applications: map[string]*Application{},
		dockerClient: dockerClient,
		// dockerEventsChannel: dockerEventsChannel,
		Emitter: emission.NewEmitter(),
	}

	apparatchick.Emitter.SetMaxListeners(MaxListeners)

	go func() {
		for evt := range ch {
			apparatchick.HandleDockerEvent(evt)
		}
	}()

	apparatchick.Emitter.EmitAsync("applications", apparatchick.applicatioNames())

	return apparatchick, nil

}

func (a *Apparatchik) GetApplicationByName(name string) (*Application, error) {
	a.Lock()
	defer a.Unlock()
	app, found := a.applications[name]
	if !found {
		return nil, ErrApplicationNotFound
	}
	return app, nil
}

func (a *Apparatchik) Stop() {
	a.Lock()
	defer a.Unlock()
	for _, application := range a.applications {
		application.TerminateApplication()
	}
	a.applications = map[string]*Application{}
}

func (a *Apparatchik) HandleDockerEvent(evt events.Message) {
	a.Lock()
	defer a.Unlock()
	for _, application := range a.applications {
		application.HandleDockerEvent(evt)
	}
}

func (p *Apparatchik) ApplicationStatus(applicatioName string) (ApplicationStatus, error) {
	app, err := p.ApplicationByName(applicatioName)
	if err != nil {
		return ApplicationStatus{}, err
	}
	return app.Status(), nil
}

func (a *Apparatchik) GetContainerIDForGoal(applicatioName, goalName string) (*string, error) {
	app, err := a.ApplicationByName(applicatioName)
	if err != nil {
		return nil, err
	}

	// TODO add GoalByName() to Application
	goal, ok := app.Goals[goalName]
	if !ok {
		return nil, errors.New("Goal not found")
	}
	return goal.ContainerId, nil
}

func (a *Apparatchik) NewApplication(name string, config *ApplicationConfiguration) (ApplicationStatus, error) {

	a.Lock()
	defer a.Unlock()

	_, found := a.applications[name]

	if found {
		return ApplicationStatus{}, ErrApplicationAlreadyExists
	}

	application := NewApplication(name, config, a.dockerClient)
	a.applications[name] = application

	a.EmitAsync("applications", a.applicatioNames())

	return application.Status(), nil
}

func (a *Apparatchik) TerminateApplication(applicationName string) error {

	a.Lock()

	application, found := a.applications[applicationName]

	if !found {
		a.Unlock()
		return ErrApplicationNotFound
	}

	delete(a.applications, applicationName)

	a.Unlock()

	application.TerminateApplication()

	a.EmitAsync("applications", a.applicatioNames())

	return nil
}

func (a *Apparatchik) applicatioNames() []string {
	names := []string{}
	for k := range a.applications {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func (a *Apparatchik) ApplicatioNames() []string {
	a.Lock()
	defer a.Unlock()
	return a.applicatioNames()
}

func (a *Apparatchik) ApplicationByName(name string) (*Application, error) {
	a.Lock()
	defer a.Unlock()

	application, ok := a.applications[name]
	if !ok {
		return nil, ErrApplicationNotFound
	}
	return application, nil
}
