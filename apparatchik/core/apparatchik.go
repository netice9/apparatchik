package core

import (
	"errors"
	"sort"
	"sync"

	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/notifier-go"
)

type Apparatchik struct {
	sync.Mutex
	*notifier.Notifier
	applications        map[string]*Application
	dockerClient        *docker.Client
	dockerEventsChannel chan *docker.APIEvents
}

func StartApparatchik(dockerClient *docker.Client) (*Apparatchik, error) {

	dockerEventsChannel := make(chan *docker.APIEvents, 20)
	err := dockerClient.AddEventListener(dockerEventsChannel)
	if err != nil {
		return nil, err
	}
	apparatchick := &Apparatchik{
		applications:        map[string]*Application{},
		dockerClient:        dockerClient,
		dockerEventsChannel: dockerEventsChannel,
		Notifier:            &notifier.Notifier{},
	}

	// call HandleDockerEvent for every new docker event
	// in a separate go-routine
	go func() {
		for evt := range dockerEventsChannel {
			apparatchick.HandleDockerEvent(evt)
		}
	}()

	apparatchick.Notify(apparatchick.applicatioNames())

	return apparatchick, nil

}

func (a *Apparatchik) Stop() {
	a.Lock()
	defer a.Unlock()
	for _, application := range a.applications {
		application.TerminateApplication()
	}
	a.applications = map[string]*Application{}
}

func (a *Apparatchik) HandleDockerEvent(evt *docker.APIEvents) {
	a.Lock()
	defer a.Unlock()
	for _, application := range a.applications {
		application.HandleDockerEvent(evt)
	}
}

func (p *Apparatchik) ApplicationStatus(applicatioName string) (*ApplicationStatus, error) {
	app, err := p.ApplicationByName(applicatioName)
	if err != nil {
		return nil, err
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

func (a *Apparatchik) NewApplication(name string, config *ApplicationConfiguration) (*ApplicationStatus, error) {

	a.Lock()
	defer a.Unlock()

	_, found := a.applications[name]

	if found {
		return nil, ErrApplicationAlreadyExists
	}

	application := NewApplication(name, config, a.dockerClient)
	a.applications[name] = application

	applicationNames := a.applicatioNames()
	a.Notify(applicationNames)

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

	applicationNames := a.applicatioNames()
	a.Notify(applicationNames)

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

func (ap *Apparatchik) GoalTransitionLog(applicationName, goalName string) ([]TransitionLogEntry, error) {
	application, err := ap.ApplicationByName(applicationName)
	if err != nil {
		return nil, err
	}

	return application.TransitionLog(goalName)
}
