package main

import (
	// "errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type TransitionLogEntry struct {
	Time   time.Time `json:"time"`
	Status string    `json:"status"`
}

type GoalConfiguration struct {
	Image         string                   `json:"image"`
	Command       []string                 `json:"command"`
	RunAfter      []string                 `json:"run_after"`
	Links         []string                 `json:"links"`
	ExtraHosts    []string                 `json:"extra_hosts"`
	Ports         []string                 `json:"ports"`
	Expose        []string                 `json:"expose"`
	Volumes       []string                 `json:"volumes"`
	Environment   map[string]string        `json:"environment"`
	Labels        map[string]string        `json:"labels"`
	LogDriver     *string                  `json:"log_driver"`
	LogConfig     map[string]string        `json:"log_config"`
	Net           *string                  `json:"net"`
	Dns           *[]string                `json:"dns"`
	CapAdd        []string                 `json:"cap_add"`
	CapDrop       []string                 `json:"cap_drop"`
	DNSSearch     []string                 `json:"dns_search"`
	Devices       []string                 `json:"devices"`
	SecurityOpt   []string                 `json:"security_opt"`
	WorkingDir    string                   `json:"working_dir"`
	Entrypoint    []string                 `json:"entrypoint"`
	User          string                   `json:"user"`
	Hostname      string                   `json:"hostname"`
	Domainname    string                   `json:"domainname"`
	MacAddress    string                   `json:"mac_address"`
	MemLimit      int64                    `json:"mem_limit"`
	MemSwapLimit  int64                    `json:"memswap_limit"`
	Privileged    bool                     `json:"privileged"`
	Restart       *string                  `json:"restart"`
	StdinOpen     bool                     `json:"stdin_open"`
	Tty           bool                     `json:"tty"`
	CpuShares     int64                    `json:"cpu_shares"`
	CpuSet        string                   `json:"cpuset"`
	ReadOnly      bool                     `json:"read_only"`
	VolumeDrvier  string                   `json:"volume_driver"`
	AuthConfig    docker.AuthConfiguration `json:"auth_config"`
	ContainerName *string                  `json:"container_name"`
	ExternalLinks []string                 `json:"external_links"`
	SmartRestart  bool                     `json:"smart_restart"`
}

type GoalStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type Goal struct {
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
	GoalEventListeners   []chan GoalEvent
	SmartRestart         bool

	CreateContainerOptions docker.CreateContainerOptions

	ContainerId *string
	ExitCode    *int

	transitionLog []TransitionLogEntry

	StatusRequest             chan chan *GoalStatus
	StartRequest              chan bool
	DockerEvent               chan *docker.APIEvents
	GoalEvent                 chan GoalEvent
	FetchImageFinished        chan *string
	RegisterGoalEventListener chan chan GoalEvent
	ApplicationStarted        chan map[string]*Goal
	TerminateRequest          chan bool

	stateLock sync.Mutex

	statsTracker *StatsTracker
}

type GoalEvent struct {
	Name  string
	Event string
}

func (goal *Goal) setCurrentStatus(status string) {
	goal.stateLock.Lock()
	for _, listener := range goal.GoalEventListeners {
		listener <- GoalEvent{Name: goal.Name, Event: status}
	}

	goal.transitionLog = append(goal.transitionLog, TransitionLogEntry{Time: time.Now(), Status: status})

	if len(goal.transitionLog) > 255 {
		goal.transitionLog = goal.transitionLog[1:]
	}

	goal.CurrentStatus = status

	goal.stateLock.Unlock()
}

func (goal *Goal) Actor() {

	goal.DockerClient.AddEventListener(goal.DockerEvent)

	defer goal.DockerClient.RemoveEventListener(goal.DockerEvent)

	go goal.FetchImage()

	for {
		select {

		case <-goal.TerminateRequest:
			if goal.ContainerId != nil {
				goal.DockerClient.RemoveContainer(docker.RemoveContainerOptions{
					ID:            *goal.ContainerId,
					RemoveVolumes: true,
					Force:         true,
				})
			}
			return

		case fetchImageFinished := <-goal.FetchImageFinished:
			if fetchImageFinished == nil {
				goal.ImageExists = true
				if goal.canRun() {
					goal.StartContainer()
				} else {
					goal.setCurrentStatus("waiting_for_dependencies")
				}
			} else {
				goal.setCurrentStatus("error: " + *fetchImageFinished)
			}
		case event := <-goal.GoalEvent:

			if _, ok := goal.RunAfterStatuses[event.Name]; ok {
				goal.RunAfterStatuses[event.Name] = event.Event
			}
			if _, ok := goal.LinksStatuses[event.Name]; ok {
				goal.LinksStatuses[event.Name] = event.Event
			}

			if goal.canRun() {
				goal.StartContainer()
			} else if goal.shouldStop() {
				goal.StopContainer()
			}

		case goals := <-goal.ApplicationStarted:
			goal.Goals = goals

			for goalName, _ := range goal.RunAfterStatuses {
				goal.Goals[goalName].RegisterGoalEventListener <- goal.GoalEvent
			}

			for goalName, _ := range goal.LinksStatuses {
				goal.Goals[goalName].RegisterGoalEventListener <- goal.GoalEvent
			}

		case goalEventListener := <-goal.RegisterGoalEventListener:
			goal.GoalEventListeners = append(goal.GoalEventListeners, goalEventListener)

		case responseChannel := <-goal.StatusRequest:
			responseChannel <- &GoalStatus{
				Name:     goal.Name,
				Status:   goal.CurrentStatus,
				ExitCode: goal.ExitCode,
			}

		case <-goal.StartRequest:

			goal.ShouldRun = true

			if goal.canRun() {
				goal.StartContainer()
			} else {
				if goal.ImageExists {
					goal.setCurrentStatus("waiting_for_dependencies")
				} else {
					goal.setCurrentStatus("fetching_image")
				}
				for name, status := range goal.RunAfterStatuses {
					if status != "not_running" {
						goal.Goals[name].Start()
					}
				}
				for name, status := range goal.LinksStatuses {
					if status != "running" {
						goal.Goals[name].Start()
					}
				}
			}

		case dockerEvent := <-goal.DockerEvent:
			goal.HandleDockerEvent(dockerEvent)
		}
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
					goal.StartContainer()
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

func (goal *Goal) findImageIdByRepoTag() (*string, error) {
	images, err := goal.DockerClient.ListImages(docker.ListImagesOptions{})

	if err != nil {
		return nil, err
	}

	for _, image := range images {
		if Contains(image.RepoTags, goal.CreateContainerOptions.Config.Image) {
			return &image.ID, nil
		}
	}
	return nil, docker.ErrNoSuchImage
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
	} else {
		return fmt.Sprintf("ap_%s_%s", applicationName, goalName)
	}
}

func (goal *Goal) StopContainer() {
	goal.setCurrentStatus("stopping_container")
	err := goal.DockerClient.StopContainer(*goal.ContainerId, 0)
	if err != nil {
		goal.setCurrentStatus("error: " + err.Error())
	}
}

func (goal *Goal) StartContainer() {

	goal.setCurrentStatus("starting")

	go func() {

		existingContainer, err := goal.findContainerIdByName(goal.CreateContainerOptions.Name)

		if err != nil {
			goal.setCurrentStatus("error: " + err.Error())
			return
		}

		if existingContainer != nil {
			err = goal.DockerClient.RemoveContainer(docker.RemoveContainerOptions{
				ID:            existingContainer.ID,
				RemoveVolumes: true,
				Force:         true,
			})
			if err != nil {
				goal.setCurrentStatus("error: " + err.Error())
				return
			}
		}

		container, err := goal.DockerClient.CreateContainer(goal.CreateContainerOptions)

		if err != nil {
			goal.setCurrentStatus("error: " + err.Error())
			return
		}

		goal.ContainerId = &container.ID

		err = goal.DockerClient.StartContainer(container.ID, nil)

		if err != nil {
			goal.setCurrentStatus("error: " + err.Error())
			return
		}

		goal.statsTracker = NewStatsTracker(*goal.ContainerId, goal.DockerClient)
	}()

}

func (goal *Goal) Start() {
	goal.StartRequest <- true
}

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

func (goal *Goal) Inspect() (*docker.Container, error) {
	if goal.ContainerId != nil {
		return goal.DockerClient.InspectContainer(*goal.ContainerId)
	} else {
		return nil, nil
	}
}

func (goal *Goal) Status() *GoalStatus {
	responseChannel := make(chan *GoalStatus)

	goal.StatusRequest <- responseChannel

	return <-responseChannel
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

func NewGoal(goalName string, applicationName string, configs map[string]GoalConfiguration, dockerClient *docker.Client) *Goal {

	config := configs[goalName]

	goal := &Goal{
		Name:                      goalName,
		ApplicationName:           applicationName,
		StatusRequest:             make(chan chan *GoalStatus),
		StartRequest:              make(chan bool),
		DockerEvent:               make(chan *docker.APIEvents, 100),
		GoalEvent:                 make(chan GoalEvent, 100),
		ApplicationStarted:        make(chan map[string]*Goal),
		RegisterGoalEventListener: make(chan chan GoalEvent),
		TerminateRequest:          make(chan bool),
		FetchImageFinished:        make(chan *string),
		DockerClient:              dockerClient,
		CurrentStatus:             "not_running",
		RunAfterStatuses:          make(map[string]string),
		LinksStatuses:             make(map[string]string),
		AuthConfig:                config.AuthConfig,
		transitionLog:             make([]TransitionLogEntry, 0),
		UpstreamGoalStatuses:      make(map[string]string),
		SmartRestart:              config.SmartRestart,
		CreateContainerOptions: docker.CreateContainerOptions{
			Name: containerName(applicationName, goalName, config.ContainerName),
			Config: &docker.Config{
				Image:        config.Image,
				Cmd:          config.Command,
				ExposedPorts: make(map[docker.Port]struct{}),
				Env:          make([]string, 0),
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
				PortBindings:   make(map[docker.Port][]docker.PortBinding),
				Binds:          make([]string, 0),
				CapAdd:         config.CapAdd,
				CapDrop:        config.CapDrop,
				DNSSearch:      config.DNSSearch,
				Devices:        make([]docker.Device, 0),
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

	go goal.Actor()

	return goal
}

func (goal *Goal) TransitionLog() []TransitionLogEntry {
	return goal.transitionLog
}

func (goal *Goal) CurrentStats() *docker.Stats {
	return goal.statsTracker.MomentaryStats()
}

func (goal *Goal) Stats(since time.Time) *Stats {
	return goal.statsTracker.CurrentStats(since)
}

func (goal *Goal) Terminate() {
	goal.TerminateRequest <- true
}

func (goal *Goal) FetchImage() {

	repo, tag := ParseRepositoryTag(goal.CreateContainerOptions.Config.Image)

	images, err := goal.DockerClient.ListImages(docker.ListImagesOptions{
		Filter: repo,
	})

	if err != nil {
		errorString := err.Error()
		goal.FetchImageFinished <- &errorString
		return
	}

	for _, image := range images {
		if Contains(image.RepoTags, goal.CreateContainerOptions.Config.Image) {
			goal.FetchImageFinished <- nil
			return
		}
	}

	opts := docker.PullImageOptions{
		Repository: repo,
		Tag:        tag,
	}

	err = goal.DockerClient.PullImage(opts, goal.AuthConfig)

	if err != nil {
		errorString := err.Error()
		goal.FetchImageFinished <- &errorString
		return
	} else {
		images, err := goal.DockerClient.ListImages(docker.ListImagesOptions{Filter: repo})

		if err != nil {
			errorString := err.Error()
			goal.FetchImageFinished <- &errorString
			return
		}

		for _, image := range images {
			if Contains(image.RepoTags, goal.CreateContainerOptions.Config.Image) {
				goal.FetchImageFinished <- nil
				return
			}
		}
		errorMessage := "could not find image"
		goal.FetchImageFinished <- &errorMessage
	}

}
