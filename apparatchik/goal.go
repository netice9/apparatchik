package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/cine"
)

type TransitionLogEntry struct {
	Time   time.Time `json:"time"`
	Status string    `json:"status"`
}

type GoalStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type Goal struct {
	cine.Actor
	application          *Application
	Name                 string
	ApplicationName      string
	DockerClient         *docker.Client
	CurrentStatus        string
	Goals                map[string]*Goal
	RunAfterStatuses     map[string]string
	LinksStatuses        map[string]string
	UpstreamGoalStatuses map[string]string
	ShouldRun            bool
	ImageExists          bool
	AuthConfig           docker.AuthConfiguration
	SmartRestart         bool

	CreateContainerOptions docker.CreateContainerOptions

	ContainerId *string
	ExitCode    *int

	transitionLog []TransitionLogEntry

	statsTracker *StatsTracker
}

type GoalEvent struct {
	Name  string
	Event string
}

func (goal *Goal) terminateGoal() {
	// TODO use goroutine?
	if goal.ContainerId != nil {
		goal.DockerClient.RemoveContainer(docker.RemoveContainerOptions{
			ID:            *goal.ContainerId,
			RemoveVolumes: true,
			Force:         true,
		})
	}
}

func (goal *Goal) SetCurrentStatus(status string) {
	cine.Cast(goal.Self(), nil, (*Goal).setCurrentStatus, status)
}

func (goal *Goal) setCurrentStatus(status string) {

	log.Info("setting current status of goal ", goal.Name, " to ", status)

	goal.transitionLog = append(goal.transitionLog, TransitionLogEntry{Time: time.Now(), Status: status})

	if len(goal.transitionLog) > 255 {
		goal.transitionLog = goal.transitionLog[1:]
	}

	goal.application.GoalStatusUpdate(goal.Name, status)

	goal.CurrentStatus = status

}

func (goal *Goal) FetchImageFailed(reason string) {
	cine.Cast(goal.Self(), nil, (*Goal).fetchImageFailed, reason)
}

func (goal *Goal) fetchImageFailed(reason string) {
	goal.setCurrentStatus("error: " + reason)
}

func (goal *Goal) FetchImageFinished() {
	cine.Cast(goal.Self(), nil, (*Goal).fetchImageFinished)
}

func (goal *Goal) fetchImageFinished() {
	goal.ImageExists = true
	if goal.canRun() {
		goal.startContainer()
	} else {
		goal.setCurrentStatus("waiting_for_dependencies")
	}
}

func (goal *Goal) shouldStop() bool {

	isRunning := goal.CurrentStatus == "running"

	if isRunning && !goal.ShouldRun {
		return true
	}

	for _, status := range goal.LinksStatuses {
		if isRunning && status != "running" {
			return true
		}
	}

	return false

}

func (goal *Goal) canRun() bool {

	if !goal.ShouldRun {
		return false
	}

	if !goal.ImageExists {
		return false
	}

	for _, status := range goal.RunAfterStatuses {
		if status != "terminated" {
			return false
		}
	}
	for _, status := range goal.LinksStatuses {
		if status != "running" {
			return false
		}
	}
	return goal.CurrentStatus == "waiting_for_dependencies" ||
		goal.CurrentStatus == "fetching_image" ||
		goal.CurrentStatus == "terminated" ||
		(goal.CurrentStatus == "failed" && goal.SmartRestart)

}

func (goal *Goal) HandleDockerEvent(evt *docker.APIEvents) {
	cine.Cast(goal.Self(), nil, (*Goal).handleDockerEvent, evt)
}

func (goal *Goal) handleDockerEvent(evt *docker.APIEvents) {
	if goal.ContainerId != nil && evt.ID == *goal.ContainerId {
		if evt.Status == "start" {
			goal.setCurrentStatus("running")
		}

		if evt.Status == "die" {

			container, err := goal.DockerClient.InspectContainer(*goal.ContainerId)
			if err != nil {
				goal.setCurrentStatus("error: " + err.Error())
			} else {
				goal.ExitCode = &container.State.ExitCode
			}
			if *goal.ExitCode == 0 {
				goal.setCurrentStatus("terminated")
			} else {
				goal.setCurrentStatus("failed")
				if goal.canRun() {
					goal.startContainer()
				}

			}
		}
	}

}

func Contains(stringSlice []string, value string) bool {
	for _, current := range stringSlice {
		if current == value {
			return true
		}
	}
	return false
}

func ContainsString(slice []string, val string) bool {
	for _, c := range slice {
		if c == val {
			return true
		}
	}
	return false
}

func (goal *Goal) findContainerIdByName(name string) (*docker.APIContainers, error) {
	containers, err := goal.DockerClient.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if ContainsString(container.Names, "/"+name) {
			return &container, nil
		}
	}
	return nil, nil
}

func containerName(applicationName string, goalName string, configName *string) string {
	if configName != nil {
		return *configName
	}
	return fmt.Sprintf("ap_%s_%s", applicationName, goalName)
}

func (goal *Goal) StopContainer() {
	goal.setCurrentStatus("stopping_container")
	err := goal.DockerClient.StopContainer(*goal.ContainerId, 0)
	if err != nil {
		goal.setCurrentStatus("error: " + err.Error())
	}
}

func (goal *Goal) startContainer() {

	goal.setCurrentStatus("starting")

	go func() {

		existingContainer, err := goal.findContainerIdByName(goal.CreateContainerOptions.Name)

		if err != nil {
			goal.SetCurrentStatus("error: " + err.Error())
			return
		}

		if existingContainer != nil {
			err = goal.DockerClient.RemoveContainer(docker.RemoveContainerOptions{
				ID:            existingContainer.ID,
				RemoveVolumes: true,
				Force:         true,
			})
			if err != nil {
				goal.SetCurrentStatus("error: " + err.Error())
				return
			}
		}

		container, err := goal.DockerClient.CreateContainer(goal.CreateContainerOptions)

		if err != nil {
			goal.SetCurrentStatus("error: " + err.Error())
			return
		}

		goal.SetContainerID(container.ID)

		err = goal.DockerClient.StartContainer(container.ID, nil)

		if err != nil {
			goal.SetCurrentStatus("error: " + err.Error())
			return
		}

		goal.ContainerStarted()

	}()

}

func (goal *Goal) ContainerStarted() {
	cine.Cast(goal.Self(), nil, (*Goal).containerStarted)
}

func (goal *Goal) containerStarted() {
	// TODO kill existing stats tracker
	goal.statsTracker = NewStatsTracker(*goal.ContainerId, goal.DockerClient)
}

func (goal *Goal) SetContainerID(containerID string) {
	cine.Cast(goal.Self(), nil, (*Goal).setContainerID, containerID)
}

func (goal *Goal) setContainerID(containerID string) {
	goal.ContainerId = &containerID
}

// TODO move up the stream - Application?, consider async
func (goal *Goal) Logs(w io.Writer) error {
	return goal.DockerClient.Logs(docker.LogsOptions{
		Container:    *goal.ContainerId,
		OutputStream: w,
		ErrorStream:  w,
		Stdout:       true,
		Stderr:       true,
		Tail:         "400",
	})
}

// TODO move this down the chain - application level
func (goal *Goal) Inspect() (*docker.Container, error) {
	if goal.ContainerId != nil {
		return goal.DockerClient.InspectContainer(*goal.ContainerId)
	} else {
		return nil, nil
	}
}

func ParseRepositoryTag(repos string) (string, string) {
	n := strings.Index(repos, "@")
	if n >= 0 {
		parts := strings.Split(repos, "@")
		return parts[0], parts[1]
	}
	n = strings.LastIndex(repos, ":")
	if n < 0 {
		return repos, ""
	}
	if tag := repos[n+1:]; !strings.Contains(tag, "/") {
		return repos[:n], tag
	}
	return repos, ""
}

func NewGoal(application *Application, goalName string, applicationName string, configs map[string]*GoalConfiguration, dockerClient *docker.Client) *Goal {

	config := configs[goalName]

	goal := &Goal{
		Actor:                cine.Actor{},
		application:          application,
		Name:                 goalName,
		ApplicationName:      applicationName,
		DockerClient:         dockerClient,
		CurrentStatus:        "not_running",
		RunAfterStatuses:     map[string]string{},
		LinksStatuses:        map[string]string{},
		AuthConfig:           config.AuthConfig,
		transitionLog:        []TransitionLogEntry{},
		UpstreamGoalStatuses: map[string]string{},
		SmartRestart:         config.SmartRestart,
		CreateContainerOptions: docker.CreateContainerOptions{
			Name: containerName(applicationName, goalName, config.ContainerName),
			Config: &docker.Config{
				Image:        config.Image,
				Cmd:          config.Command,
				ExposedPorts: map[docker.Port]struct{}{},
				Env:          []string{},
				Labels:       config.Labels,
				WorkingDir:   config.WorkingDir,
				Entrypoint:   config.Entrypoint,
				User:         config.User,
				Hostname:     config.Hostname,
				Domainname:   config.Domainname,
				MacAddress:   config.MacAddress,
				AttachStdin:  config.StdinOpen,
				Tty:          config.Tty,
				VolumeDriver: config.VolumeDrvier,
			},
			HostConfig: &docker.HostConfig{
				ExtraHosts:     config.ExtraHosts,
				PortBindings:   map[docker.Port][]docker.PortBinding{},
				Binds:          []string{},
				CapAdd:         config.CapAdd,
				CapDrop:        config.CapDrop,
				DNSSearch:      config.DNSSearch,
				Devices:        []docker.Device{},
				SecurityOpt:    config.SecurityOpt,
				Memory:         config.MemLimit,
				MemorySwap:     config.MemSwapLimit,
				Privileged:     config.Privileged,
				CPUShares:      config.CpuShares,
				CPUSetCPUs:     config.CpuSet,
				CPUSetMEMs:     config.CpuSet,
				ReadonlyRootfs: config.ReadOnly,
			},
		},
	}

	if config.Restart != nil {
		goal.CreateContainerOptions.HostConfig.RestartPolicy = docker.RestartPolicy{Name: *config.Restart}
	}

	for _, deviceString := range config.Devices {
		parts := strings.Split(deviceString, ":")
		perm := "mrw"
		hostDevice := parts[0]
		containerDevice := parts[0]
		if len(parts) == 3 {
			perm = parts[2]
			containerDevice = parts[1]
		} else if len(parts) == 2 {
			if len(parts[1]) > 3 {
				containerDevice = parts[1]
			} else {
				containerDevice = parts[0]
				perm = parts[1]
			}
		}

		goal.CreateContainerOptions.HostConfig.Devices = append(goal.CreateContainerOptions.HostConfig.Devices, docker.Device{
			PathOnHost:        hostDevice,
			PathInContainer:   containerDevice,
			CgroupPermissions: perm})
	}

	if config.Dns != nil {
		goal.CreateContainerOptions.HostConfig.DNS = *config.Dns
	}

	if config.Net != nil {
		goal.CreateContainerOptions.HostConfig.NetworkMode = *config.Net
	}

	if config.LogDriver != nil {
		goal.CreateContainerOptions.HostConfig.LogConfig = docker.LogConfig{
			Type:   *config.LogDriver,
			Config: config.LogConfig,
		}
	}

	for k, v := range config.Environment {
		goal.CreateContainerOptions.Config.Env = append(goal.CreateContainerOptions.Config.Env, k+"="+v)
	}

	for _, bind := range config.Volumes {
		parts := strings.Split(bind, ":")
		if len(parts) == 1 {
			goal.CreateContainerOptions.HostConfig.Binds = append(goal.CreateContainerOptions.HostConfig.Binds, parts[0]+":"+parts[0])
		} else if len(parts) == 2 {
			if parts[1] == "rw" || parts[1] == "ro" {
				goal.CreateContainerOptions.HostConfig.Binds = append(goal.CreateContainerOptions.HostConfig.Binds, parts[0]+":"+parts[0]+":"+parts[1])
			} else {
				goal.CreateContainerOptions.HostConfig.Binds = append(goal.CreateContainerOptions.HostConfig.Binds, bind)
			}
		} else {
			goal.CreateContainerOptions.HostConfig.Binds = append(goal.CreateContainerOptions.HostConfig.Binds, bind)
		}

	}

	for _, name := range config.RunAfter {
		goal.RunAfterStatuses[name] = "unknown"
	}

	for _, link := range config.Links {
		parts := strings.Split(link, ":")
		name := parts[0]
		alias := name
		if len(parts) > 1 {
			alias = parts[1]
		}
		goal.LinksStatuses[name] = "unknown"

		goal.CreateContainerOptions.HostConfig.Links = append(goal.CreateContainerOptions.HostConfig.Links, containerName(applicationName, name, configs[name].ContainerName)+":"+alias)

	}

	for _, link := range config.ExternalLinks {
		parts := strings.Split(link, ":")
		name := parts[0]
		alias := name
		if len(parts) > 1 {
			alias = parts[1]
		}

		goal.CreateContainerOptions.HostConfig.Links = append(goal.CreateContainerOptions.HostConfig.Links, name+":"+alias)

	}

	for _, port := range config.Ports {
		protoParts := strings.Split(port, "/")

		proto := "tcp"

		if len(protoParts) == 2 {
			proto = protoParts[1]
		}

		parts := strings.Split(protoParts[0], ":")

		hostPort := ""

		containerPort := parts[0]

		if len(parts) == 2 {
			hostPort = parts[0]
			containerPort = parts[1]
			portBinding := docker.PortBinding{HostPort: hostPort}
			goal.CreateContainerOptions.HostConfig.PortBindings[docker.Port(containerPort+"/"+proto)] = []docker.PortBinding{portBinding}
		} else {
			goal.CreateContainerOptions.HostConfig.PortBindings[docker.Port(containerPort+"/"+proto)] = []docker.PortBinding{}
		}

	}

	for _, port := range config.Expose {
		protoParts := strings.Split(port, "/")

		proto := "tcp"

		if len(protoParts) == 2 {
			proto = protoParts[1]
		}

		goal.CreateContainerOptions.Config.ExposedPorts[docker.Port(protoParts[0]+"/"+proto)] = struct{}{}

	}

	cine.StartActor(goal)
	cine.Cast(goal.Self(), nil, (*Goal).fetchImage)

	return goal
}

func (goal *Goal) TransitionLog() []TransitionLogEntry {
	result, err := cine.Call(goal.Self(), (*Goal).getTransitionLog)
	if err != nil {
		panic(err)
	}

	return result[0].([]TransitionLogEntry)
}

func (goal *Goal) getTransitionLog() []TransitionLogEntry {
	return goal.transitionLog
}

// TODO move to actor
func (goal *Goal) CurrentStats() *docker.Stats {
	return goal.statsTracker.MomentaryStats()
}

// TODO move to actor
func (goal *Goal) Stats(since time.Time) *Stats {
	return goal.statsTracker.CurrentStats(since)
}

func (goal *Goal) fetchImage() {

	repo, tag := ParseRepositoryTag(goal.CreateContainerOptions.Config.Image)

	go func() {

		images, err := goal.DockerClient.ListImages(docker.ListImagesOptions{
			Filter: repo,
		})

		if err != nil {
			goal.FetchImageFailed(err.Error())
			return
		}

		for _, image := range images {
			if Contains(image.RepoTags, goal.CreateContainerOptions.Config.Image) {
				goal.fetchImageFinished()
				return
			}
		}

		opts := docker.PullImageOptions{
			Repository: repo,
			Tag:        tag,
		}

		err = goal.DockerClient.PullImage(opts, goal.AuthConfig)

		if err != nil {
			goal.FetchImageFailed(err.Error())
			return
		}
		images, err = goal.DockerClient.ListImages(docker.ListImagesOptions{Filter: repo})

		if err != nil {
			goal.FetchImageFailed(err.Error())
			return
		}

		for _, image := range images {
			if Contains(image.RepoTags, goal.CreateContainerOptions.Config.Image) {
				goal.fetchImageFinished()
				return
			}
		}
		errorMessage := "could not find image"
		goal.FetchImageFailed(errorMessage)

	}()

}

func (goal *Goal) SiblingStatusUpdate(goalName, status string) {
	cine.Cast(goal.Self(), nil, (*Goal).siblingStatusUpdate, goalName, status)
}

func (goal *Goal) siblingStatusUpdate(goalName, status string) {
	if _, ok := goal.RunAfterStatuses[goalName]; ok {
		goal.RunAfterStatuses[goalName] = status
	}
	if _, ok := goal.LinksStatuses[goalName]; ok {
		goal.LinksStatuses[goalName] = status
	}

	if goal.canRun() {
		goal.startContainer()
	} else if goal.shouldStop() {
		goal.StopContainer()
	}
}

func (goal *Goal) Status() *GoalStatus {
	result, err := cine.Call(goal.Self(), (*Goal).status)
	if err != nil {
		panic(err)
	}
	return result[0].(*GoalStatus)
}

func (goal *Goal) status() *GoalStatus {
	return &GoalStatus{
		Name:     goal.Name,
		Status:   goal.CurrentStatus,
		ExitCode: goal.ExitCode,
	}
}

func (goal *Goal) Start() {
	cine.Cast(goal.Self(), nil, (*Goal).start)
}

func (goal *Goal) start() {
	goal.ShouldRun = true

	if goal.canRun() {
		goal.startContainer()
	} else {
		if goal.ImageExists {
			goal.setCurrentStatus("waiting_for_dependencies")
		} else {
			goal.setCurrentStatus("fetching_image")
		}
		for name, status := range goal.RunAfterStatuses {
			if status != "not_running" {
				goal.application.RequestGoalStart(name)
			}
		}
		for name, status := range goal.LinksStatuses {
			if status != "running" {
				goal.application.RequestGoalStart(name)
			}
		}
	}

}

func (goal *Goal) Terminate(errReason error) {
	goal.terminateGoal()

	// TODO stop tracker

}

func (goal *Goal) TerminateGoal() {
	cine.Stop(goal.Self())
}
