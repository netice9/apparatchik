package core

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/draganm/emission"
	"github.com/netice9/apparatchik/core/stats"
)

const trackerHistorySize = 120

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
	application          *Application
	Name                 string
	ApplicationName      string
	DockerClient         *client.Client
	CurrentStatus        string
	Goals                map[string]*Goal
	RunAfterStatuses     map[string]string
	LinksStatuses        map[string]string
	UpstreamGoalStatuses map[string]string
	ShouldRun            bool
	ImageExists          bool
	AuthConfig           AuthConfiguration
	SmartRestart         bool

	containerName    string
	containerConfig  *container.Config
	hostConfig       *container.HostConfig
	networkingConfig *network.NetworkingConfig

	ContainerId *string
	ExitCode    *int

	*emission.Emitter

	tail    []string
	tracker *stats.Tracker
}

type GoalEvent struct {
	Name  string
	Event string
}

func (g *Goal) Tail() string {
	g.Lock()
	defer g.Unlock()
	return strings.Join(g.tail, "")
}

func (g *Goal) AddLineToTail(line string) {
	g.Lock()
	defer g.Unlock()

	if len(g.tail) > 400 {
		g.tail = g.tail[1:]
	}

	g.tail = append(g.tail, line)
	g.EmitAsync("tail", strings.Join(g.tail, ""))
}

func (goal *Goal) TerminateGoal() {
	if goal.ContainerId != nil {
		containerID := *goal.ContainerId
		err := goal.DockerClient.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true})
		if err != nil {
			log.Error(err)
		}
	}
	goal.EmitAsync("terminated")
}

func (goal *Goal) SetCurrentStatus(status string) {
	goal.Lock()
	defer goal.Unlock()
	goal.setCurrentStatus(status)
	goal.broadcastStatus()
}

func (goal *Goal) setCurrentStatus(status string) {

	log.Debug("setting current status of goal ", goal.Name, " to ", status)

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

func (goal *Goal) HandleDockerEvent(evt events.Message) {
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

func (goal *Goal) startTailingLog() {

	goal.AddLineToTail("----------\n")
	goal.AddLineToTail(fmt.Sprintf("Container with ID %q started\n", *goal.ContainerId))
	rc, err := goal.DockerClient.ContainerLogs(context.Background(), *goal.ContainerId, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
		Details:    false,
	})
	if err != nil {
		goal.AddLineToTail("Could not tail output: " + err.Error() + "\n")
	}
	br := bufio.NewReader(rc)
	defer rc.Close()
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			goal.AddLineToTail("output closed...\n")
			return
		}
		goal.AddLineToTail(line[8:])
	}

}

func (goal *Goal) startTrackingContainer() {

	stream, err := goal.DockerClient.ContainerStats(context.Background(), *goal.ContainerId, true)
	if err != nil {
		log.Error(err)
		return
	}

	decoder := json.NewDecoder(stream.Body)

	stats := types.StatsJSON{}

	for {
		err = decoder.Decode(&stats)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Error(err)
			return
		}

		goal.AddStats(stats)
	}

}

func (goal *Goal) AddStats(st types.StatsJSON) {
	goal.Lock()
	defer goal.Unlock()

	goal.tracker.Add(stats.Entry{
		Time:   st.Read,
		CPU:    st.CPUStats.CPUUsage.TotalUsage - st.PreCPUStats.CPUUsage.TotalUsage,
		Memory: st.MemoryStats.Usage,
	})
	goal.EmitAsync("stats", goal.tracker.Entries())
}

func (goal *Goal) handleDockerEvent(evt events.Message) {
	if goal.ContainerId != nil && evt.ID == *goal.ContainerId {
		if evt.Status == "start" {
			goal.setCurrentStatus("running")

			go goal.startTailingLog()
			go goal.startTrackingContainer()

		}

		if evt.Status == "die" {
			containerID := *goal.ContainerId
			go func() {
				container, err := goal.DockerClient.ContainerInspect(context.Background(), containerID)
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

func (goal *Goal) findContainerIdByName(name string) (*types.Container, error) {
	// containers, err := goal.DockerClient.ListContainers(docker.ListContainersOptions{All: true})
	containers, err := goal.DockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
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
		err := goal.DockerClient.ContainerStop(context.Background(), *goal.ContainerId, nil)
		if err != nil {
			goal.SetCurrentStatus("error: " + err.Error())
		}
	}()
}

func (goal *Goal) startContainer() {

	goal.setCurrentStatus("starting")

	go func() {

		existingContainer, err := goal.findContainerIdByName(goal.containerName)

		if err != nil {
			goal.SetCurrentStatus("error: " + err.Error())
			return
		}

		if existingContainer != nil {
			err = goal.DockerClient.ContainerRemove(context.Background(), existingContainer.ID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
			if err != nil {
				goal.SetCurrentStatus("error: " + err.Error())
				return
			}
		}

		container, err := goal.DockerClient.ContainerCreate(context.Background(), goal.containerConfig, goal.hostConfig, goal.networkingConfig, goal.containerName)

		if err != nil {
			goal.SetCurrentStatus("error: " + err.Error())
			return
		}

		goal.SetContainerID(container.ID)

		// err = goal.DockerClient.StartContainer(container.ID, nil)
		err = goal.DockerClient.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{})

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

// TODO move this down the chain - application level
func (goal *Goal) Inspect() (types.ContainerJSON, error) {
	containerID := goal.GetContainerID()
	if containerID != nil {
		// return goal.DockerClient.InspectContainer(*containerID)
		return goal.DockerClient.ContainerInspect(context.Background(), *containerID)
	}
	return types.ContainerJSON{}, errors.New("Conainer is not running")
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

func NewGoal(application *Application, goalName string, applicationName string, configs map[string]*GoalConfiguration, dockerClient *client.Client) *Goal {

	config := configs[goalName]

	emitter := emission.NewEmitter()
	emitter.SetMaxListeners(MaxListeners)

	goal := &Goal{
		application:          application,
		Name:                 goalName,
		ApplicationName:      applicationName,
		DockerClient:         dockerClient,
		CurrentStatus:        "not_running",
		RunAfterStatuses:     map[string]string{},
		LinksStatuses:        map[string]string{},
		AuthConfig:           config.AuthConfig,
		UpstreamGoalStatuses: map[string]string{},
		SmartRestart:         config.SmartRestart,
		containerConfig: &container.Config{
			Image:        config.Image,
			Cmd:          config.Command,
			ExposedPorts: map[nat.Port]struct{}{},
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
			// TODO this has changed!
			// VolumeDriver: config.VolumeDriver,
			AttachStdin:  config.AttachStdin,
			AttachStdout: config.AttachStdout,
			AttachStderr: config.AttachStderr,
		},
		hostConfig: &container.HostConfig{
			ExtraHosts:     config.ExtraHosts,
			PortBindings:   nat.PortMap{},
			Binds:          []string{},
			CapAdd:         config.CapAdd,
			CapDrop:        config.CapDrop,
			DNSSearch:      config.DNSSearch,
			SecurityOpt:    config.SecurityOpt,
			Privileged:     config.Privileged,
			ReadonlyRootfs: config.ReadOnly,
			Resources: container.Resources{
				Devices:    []container.DeviceMapping{},
				Memory:     config.MemLimit,
				MemorySwap: config.MemSwapLimit,
				CPUShares:  config.CpuShares,
				CpusetCpus: config.CpuSet,
				CpusetMems: config.CpuSet,
			},
		},
		networkingConfig: &network.NetworkingConfig{},
		Emitter:          emitter,
		tracker:          stats.NewTracker(120 * time.Second),
	}

	if config.Restart != "" {
		// goal.hostConfig.RestartPolicy = docker.RestartPolicy{Name: config.Restart}
		goal.hostConfig.RestartPolicy.Name = config.Restart
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

		goal.hostConfig.Devices = append(goal.hostConfig.Devices, container.DeviceMapping{
			PathOnHost:        hostDevice,
			PathInContainer:   containerDevice,
			CgroupPermissions: perm,
		})
	}

	if len(config.Dns) != 0 {
		goal.hostConfig.DNS = config.Dns
	}

	if config.Net != "" {
		goal.hostConfig.NetworkMode = container.NetworkMode(config.Net)
	}

	if config.LogDriver != "" {
		goal.hostConfig.LogConfig = container.LogConfig{
			Type:   config.LogDriver,
			Config: config.LogConfig,
		}
	}

	for k, v := range config.Environment {
		goal.containerConfig.Env = append(goal.containerConfig.Env, k+"="+v)
	}

	for _, bind := range config.Volumes {
		parts := strings.Split(bind, ":")
		if len(parts) == 1 {
			goal.hostConfig.Binds = append(goal.hostConfig.Binds, replaceRelativePath(parts[0]+":"+parts[0]))
		} else if len(parts) == 2 {
			if parts[1] == "rw" || parts[1] == "ro" {
				goal.hostConfig.Binds = append(goal.hostConfig.Binds, replaceRelativePath(parts[0]+":"+parts[0]+":"+parts[1]))
			} else {
				goal.hostConfig.Binds = append(goal.hostConfig.Binds, replaceRelativePath(bind))
			}
		} else {
			goal.hostConfig.Binds = append(goal.hostConfig.Binds, replaceRelativePath(bind))
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

		goal.hostConfig.Links = append(goal.hostConfig.Links, containerName(applicationName, name, configs[name].ContainerName)+":"+alias)

	}

	for _, link := range config.ExternalLinks {
		parts := strings.Split(link, ":")
		name := parts[0]
		alias := name
		if len(parts) > 1 {
			alias = parts[1]
		}

		goal.hostConfig.Links = append(goal.hostConfig.Links, name+":"+alias)

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
			portBinding := nat.PortBinding{HostPort: hostPort}
			goal.hostConfig.PortBindings[nat.Port(containerPort+"/"+proto)] = []nat.PortBinding{portBinding}
		} else {
			goal.hostConfig.PortBindings[nat.Port(containerPort+"/"+proto)] = []nat.PortBinding{}
		}

	}

	for _, port := range config.Expose {
		protoParts := strings.Split(port, "/")

		proto := "tcp"

		if len(protoParts) == 2 {
			proto = protoParts[1]
		}

		goal.containerConfig.ExposedPorts[nat.Port(protoParts[0]+"/"+proto)] = struct{}{}

	}

	goal.containerName = fmt.Sprintf("ap_%s_%s", applicationName, goalName)
	if config.ContainerName != "" {
		goal.containerName = config.ContainerName
	}

	goal.FetchImage()

	goal.broadcastStatus()

	return goal
}

func (g *Goal) broadcastStatus() {
	g.Emitter.EmitAsync("update", g.status())
}

func replaceRelativePath(pth string) string {
	if strings.HasPrefix(pth, "./") {
		wd, _ := os.Getwd()
		return path.Join(wd, pth[2:])
	}
	return pth
}

func (goal *Goal) FetchImage() {

	goal.Lock()
	defer goal.Unlock()

	go func() {

		// _, _, err := goal.DockerClient.ImageInspectWithRaw(context.Background(), goal.containerConfig.Image)
		//
		// if err != nil && !client.IsErrImageNotFound(err) {
		// 	log.Error(err)
		// 	goal.FetchImageFailed(err.Error())
		// 	return
		// }
		//
		// if err == nil {
		// 	goal.FetchImageFinished()
		// 	return
		// }

		r, err := goal.DockerClient.ImagePull(context.Background(), goal.containerConfig.Image, types.ImagePullOptions{
			RegistryAuth: goal.AuthConfig.toDockerAuthConfig(),
		})

		if err != nil {
			goal.FetchImageFailed(err.Error())
			return
		}

		_, err = io.Copy(ioutil.Discard, r)
		if err != nil {
			goal.FetchImageFailed(err.Error())
			return
		}

		err = r.Close()

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
