package core

//go:generate gospecific -pkg=github.com/netice9/notifier-go -specific-type=GoalStatus -out-dir=.
//go:generate mv notifier.go goal_notifier.go
//go:generate sed -i "s/^package notifier/package core/" goal_notifier.go
//go:generate sed -i "s/Notifier/GoalNotifier/g" goal_notifier.go

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

const trackerHistorySize = 120
const transitionLogMaxLength = 255

type Sample struct {
	Value uint64    `json:"value"`
	Time  time.Time `json:"time"`
}

type Stats struct {
	CpuStats []Sample `json:"cpu_stats"`
	MemStats []Sample `json:"mem_stats"`
}

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
	sync.Mutex
	*GoalNotifier
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
	AuthConfig           AuthConfiguration
	SmartRestart         bool

	CreateContainerOptions docker.CreateContainerOptions

	ContainerId *string
	ExitCode    *int

	transitionLog []TransitionLogEntry

	// statsTracker *StatsTracker

	stats      Stats
	lastSample *docker.Stats
}

type GoalEvent struct {
	Name  string
	Event string
}

func (goal *Goal) TerminateGoal() {
	if goal.ContainerId != nil {
		containerID := *goal.ContainerId
		goal.DockerClient.RemoveContainer(docker.RemoveContainerOptions{
			ID:            containerID,
			RemoveVolumes: true,
			Force:         true,
		})
	}
	goal.GoalNotifier.Close()
}

func (goal *Goal) SetCurrentStatus(status string) {
	goal.Lock()
	defer goal.Unlock()
	goal.setCurrentStatus(status)
	goal.broadcastStatus()
}

func (goal *Goal) setCurrentStatus(status string) {

	log.Debug("setting current status of goal ", goal.Name, " to ", status)

	goal.transitionLog = append(goal.transitionLog, TransitionLogEntry{Time: time.Now(), Status: status})

	if len(goal.transitionLog) > transitionLogMaxLength {
		goal.transitionLog = goal.transitionLog[1:]
	}

	go goal.application.GoalStatusUpdate(goal.Name, status)

	goal.CurrentStatus = status

}

func (goal *Goal) FetchImageFailed(reason string) {
	goal.SetCurrentStatus("error: " + reason)
}

func (goal *Goal) FetchImageFinished() {
	goal.Lock()
	defer goal.Unlock()
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
		// goal.CurrentStatus == "terminated" ||
		(goal.CurrentStatus == "failed" && goal.SmartRestart)

}

func (goal *Goal) HandleDockerEvent(evt *docker.APIEvents) {
	goal.Lock()
	defer goal.Unlock()
	goal.handleDockerEvent(evt)
}

func (goal *Goal) SetExitCode(exitCode int) {
	goal.Lock()
	defer goal.Unlock()
	goal.ExitCode = &exitCode
	if exitCode == 0 {
		goal.setCurrentStatus("terminated")
	} else {
		goal.setCurrentStatus("failed")
		if goal.canRun() {
			goal.startContainer()
		}
	}

}
func (goal *Goal) handleDockerEvent(evt *docker.APIEvents) {
	if goal.ContainerId != nil && evt.ID == *goal.ContainerId {
		if evt.Status == "start" {
			goal.setCurrentStatus("running")
		}

		if evt.Status == "die" {
			containerID := *goal.ContainerId
			go func() {
				container, err := goal.DockerClient.InspectContainer(containerID)
				if err != nil {
					goal.SetCurrentStatus("error: " + err.Error())
					return
				}
				goal.SetExitCode(container.State.ExitCode)

			}()
		}
	}

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

func containerName(applicationName string, goalName string, configName string) string {
	if configName != "" {
		return configName
	}
	return fmt.Sprintf("ap_%s_%s", applicationName, goalName)
}

func (goal *Goal) StopContainer() {
	goal.SetCurrentStatus("stopping_container")
	go func() {
		err := goal.DockerClient.StopContainer(*goal.ContainerId, 0)
		if err != nil {
			goal.SetCurrentStatus("error: " + err.Error())
		}
	}()
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

	goal.Lock()
	defer goal.Unlock()

	ch := make(chan *docker.Stats)

	go func() {
		go func() {
			for stat := range ch {
				goal.HandleStatsEvent(stat)
			}
		}()

		goal.DockerClient.Stats(docker.StatsOptions{
			ID:     *goal.ContainerId,
			Stats:  ch,
			Stream: true,
		})
	}()

}

func (goal *Goal) SetContainerID(containerID string) {
	goal.Lock()
	defer goal.Unlock()
	goal.ContainerId = &containerID
}

func (goal *Goal) GetContainerID() *string {
	goal.Lock()
	defer goal.Unlock()
	return goal.ContainerId
}

// TODO move up the stream - Application?, consider async
func (goal *Goal) Logs(w io.Writer) error {
	return goal.DockerClient.Logs(docker.LogsOptions{
		Container:    *goal.GetContainerID(),
		OutputStream: w,
		ErrorStream:  w,
		Stdout:       true,
		Stderr:       true,
		Tail:         "400",
	})
}

// TODO move this down the chain - application level
func (goal *Goal) Inspect() (*docker.Container, error) {
	containerID := goal.GetContainerID()
	if containerID != nil {
		return goal.DockerClient.InspectContainer(*containerID)
	}
	return nil, errors.New("Conainer is not running")

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
		application:          application,
		GoalNotifier:         NewGoalNotifier(GoalStatus{}),
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
				OpenStdin:    config.StdinOpen,
				Tty:          config.Tty,
				VolumeDriver: config.VolumeDrvier,
				AttachStdin:  config.AttachStdin,
				AttachStdout: config.AttachStdout,
				AttachStderr: config.AttachStderr,
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

	if config.Restart != "" {
		goal.CreateContainerOptions.HostConfig.RestartPolicy = docker.RestartPolicy{Name: config.Restart}
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

	if len(config.Dns) != 0 {
		goal.CreateContainerOptions.HostConfig.DNS = config.Dns
	}

	if config.Net != "" {
		goal.CreateContainerOptions.HostConfig.NetworkMode = config.Net
	}

	if config.LogDriver != "" {
		goal.CreateContainerOptions.HostConfig.LogConfig = docker.LogConfig{
			Type:   config.LogDriver,
			Config: config.LogConfig,
		}
	}

	for k, v := range config.Environment {
		goal.CreateContainerOptions.Config.Env = append(goal.CreateContainerOptions.Config.Env, k+"="+v)
	}

	for _, bind := range config.Volumes {
		parts := strings.Split(bind, ":")
		if len(parts) == 1 {
			goal.CreateContainerOptions.HostConfig.Binds = append(goal.CreateContainerOptions.HostConfig.Binds, replaceRelativePath(parts[0]+":"+parts[0]))
		} else if len(parts) == 2 {
			if parts[1] == "rw" || parts[1] == "ro" {
				goal.CreateContainerOptions.HostConfig.Binds = append(goal.CreateContainerOptions.HostConfig.Binds, replaceRelativePath(parts[0]+":"+parts[0]+":"+parts[1]))
			} else {
				goal.CreateContainerOptions.HostConfig.Binds = append(goal.CreateContainerOptions.HostConfig.Binds, replaceRelativePath(bind))
			}
		} else {
			goal.CreateContainerOptions.HostConfig.Binds = append(goal.CreateContainerOptions.HostConfig.Binds, replaceRelativePath(bind))
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

	goal.FetchImage()

	goal.broadcastStatus()

	return goal
}

func (g *Goal) broadcastStatus() {
	g.Notify(g.status())
}

func replaceRelativePath(pth string) string {
	if strings.HasPrefix(pth, "./") {
		wd, _ := os.Getwd()
		return path.Join(wd, pth[2:])
	}
	return pth
}

func (goal *Goal) GetTransitionLog() []TransitionLogEntry {
	goal.Lock()
	defer goal.Unlock()
	log := make([]TransitionLogEntry, len(goal.transitionLog))
	copy(log, goal.transitionLog)
	return log
}

func (goal *Goal) CurrentStats() *docker.Stats {
	goal.Lock()
	defer goal.Unlock()
	return goal.lastSample
}

func (goal *Goal) Stats(since time.Time) *Stats {
	goal.Lock()
	defer goal.Unlock()
	return &Stats{
		CpuStats: LimitSampleByTime(goal.stats.CpuStats, since),
		MemStats: LimitSampleByTime(goal.stats.MemStats, since),
	}
}

func (goal *Goal) FetchImage() {

	goal.Lock()
	defer goal.Unlock()

	repo, tag := ParseRepositoryTag(goal.CreateContainerOptions.Config.Image)

	go func() {

		_, err := goal.DockerClient.InspectImage(goal.CreateContainerOptions.Config.Image)

		if err != nil && err != docker.ErrNoSuchImage {
			goal.FetchImageFailed(err.Error())
			return
		}

		if err == nil {
			goal.FetchImageFinished()
			return
		}

		opts := docker.PullImageOptions{
			Repository: repo,
			Tag:        tag,
		}

		err = goal.DockerClient.PullImage(opts, goal.AuthConfig.toDockerAuthConfig())

		if err != nil {
			goal.FetchImageFailed(err.Error())
			return
		}

		goal.FetchImageFinished()

	}()

}

func (goal *Goal) SiblingStatusUpdate(goalName, status string) {
	goal.Lock()
	defer goal.Unlock()
	if _, ok := goal.RunAfterStatuses[goalName]; ok {
		goal.RunAfterStatuses[goalName] = status
	}
	if _, ok := goal.LinksStatuses[goalName]; ok {
		goal.LinksStatuses[goalName] = status
	}

	if goal.canRun() {
		goal.startContainer()
	} else if goal.shouldStop() {
		go goal.StopContainer()
	}
}

func (goal *Goal) status() GoalStatus {
	return GoalStatus{
		Name:     goal.Name,
		Status:   goal.CurrentStatus,
		ExitCode: goal.ExitCode,
	}
}

func (goal *Goal) Status() GoalStatus {
	goal.Lock()
	defer goal.Unlock()
	return goal.status()
}

func (goal *Goal) Start() {
	goal.Lock()
	defer goal.Unlock()
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
				go goal.application.RequestGoalStart(name)
			}
		}
		for name, status := range goal.LinksStatuses {
			if status != "running" {
				go goal.application.RequestGoalStart(name)
			}
		}
	}

}

func (goal *Goal) HandleStatsEvent(stats *docker.Stats) {

	goal.Lock()
	defer goal.Unlock()

	if goal.lastSample == nil {
		goal.lastSample = stats
	} else {
		cpuDiff := stats.CPUStats.CPUUsage.TotalUsage - goal.lastSample.CPUStats.CPUUsage.TotalUsage
		memory := stats.MemoryStats.Usage

		goal.stats.CpuStats = append(goal.stats.CpuStats, Sample{Value: cpuDiff, Time: stats.Read})
		if len(goal.stats.CpuStats) > trackerHistorySize {
			goal.stats.CpuStats = goal.stats.CpuStats[1:]
		}

		goal.stats.MemStats = append(goal.stats.MemStats, Sample{Value: memory, Time: stats.Read})
		if len(goal.stats.MemStats) > trackerHistorySize {
			goal.stats.MemStats = goal.stats.MemStats[1:]
		}
		goal.lastSample = stats
	}

}

func LimitSampleByTime(samples []Sample, since time.Time) []Sample {
	result := []Sample{}
	for _, sample := range samples {
		if sample.Time.After(since) {
			result = append(result, sample)
		}
	}
	return result
}
