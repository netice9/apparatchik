package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

type ApplicationConfiguration struct {
	Goals    map[string]*GoalConfiguration `json:"goals"`
	MainGoal string                        `json:"main_goal"`
}

type GoalConfiguration struct {
	Image         string                   `json:"image"`
	Command       []string                 `json:"command,omitempty"`
	RunAfter      []string                 `json:"run_after,omitempty"`
	Links         []string                 `json:"links,omitempty"`
	ExtraHosts    []string                 `json:"extra_hosts,omitempty"`
	Ports         []string                 `json:"ports,omitempty"`
	Expose        []string                 `json:"expose,omitempty"`
	Volumes       []string                 `json:"volumes,omitempty"`
	Environment   map[string]string        `json:"environment,omitempty"`
	Labels        map[string]string        `json:"labels,omitempty"`
	LogDriver     *string                  `json:"log_driver,omitempty"`
	LogConfig     map[string]string        `json:"log_config,omitempty"`
	Net           *string                  `json:"net,omitempty"`
	Dns           *[]string                `json:"dns,omitempty"`
	CapAdd        []string                 `json:"cap_add,omitempty"`
	CapDrop       []string                 `json:"cap_drop,omitempty"`
	DNSSearch     []string                 `json:"dns_search,omitempty"`
	Devices       []string                 `json:"devices,omitempty"`
	SecurityOpt   []string                 `json:"security_opt,omitempty"`
	WorkingDir    string                   `json:"working_dir,omitempty"`
	Entrypoint    []string                 `json:"entrypoint,omitempty"`
	User          string                   `json:"user,omitempty"`
	Hostname      string                   `json:"hostname,omitempty"`
	Domainname    string                   `json:"domainname,omitempty"`
	MacAddress    string                   `json:"mac_address,omitempty"`
	MemLimit      int64                    `json:"mem_limit,omitempty"`
	MemSwapLimit  int64                    `json:"memswap_limit,omitempty"`
	Privileged    bool                     `json:"privileged,omitempty"`
	Restart       *string                  `json:"restart,omitempty"`
	StdinOpen     bool                     `json:"stdin_open,omitempty"`
	Tty           bool                     `json:"tty,omitempty"`
	CpuShares     int64                    `json:"cpu_shares,omitempty"`
	CpuSet        string                   `json:"cpuset,omitempty"`
	ReadOnly      bool                     `json:"read_only,omitempty"`
	VolumeDrvier  string                   `json:"volume_driver,omitempty"`
	AuthConfig    docker.AuthConfiguration `json:"auth_config,omitempty"`
	ContainerName *string                  `json:"container_name,omitempty"`
	ExternalLinks []string                 `json:"external_links,omitempty"`
	SmartRestart  bool                     `json:"smart_restart,omitempty"`
}

func (gc *GoalConfiguration) Clone() *GoalConfiguration {
	copy := *gc
	return &copy
}

type LinkedContainer struct {
	Name  string
	Alias string
}

func (gc *GoalConfiguration) LinkedContainers() []LinkedContainer {

	result := []LinkedContainer{}

	for _, link := range gc.Links {

		parts := strings.SplitN(link, ":", 2)
		lc := LinkedContainer{parts[0], parts[0]}
		if len(parts) == 2 {
			lc.Alias = parts[1]
		}
		result = append(result, lc)
	}

	return result

}

func (config *ApplicationConfiguration) Clone() *ApplicationConfiguration {
	clone := *config
	clone.Goals = map[string]*GoalConfiguration{}
	for goalName, goal := range config.Goals {

		clone.Goals[goalName] = goal.Clone()
	}
	return &clone
}

var goalNameExpression = regexp.MustCompile("^[0-9a-zA-Z_\\.\\-]+$")

var imageExpression = regexp.MustCompile("^[0-9a-zA-Z\\.\\-/:_]+:[0-9a-zA-Z\\.\\-_]+$")

func (config *ApplicationConfiguration) Validate() error {
	if config.MainGoal == "" {
		return errors.New("Main goal is not set")
	}
	if _, ok := config.Goals[config.MainGoal]; !ok {
		return errors.New(fmt.Sprintf("Main goal '%s' is not defined", config.MainGoal))
	}
	for name, goal := range config.Goals {
		if !goalNameExpression.MatchString(name) {
			return errors.New(fmt.Sprintf("Goal '%s' has invalid name", name))
		}
		if !imageExpression.MatchString(goal.Image) {
			return errors.New(fmt.Sprintf("Goal '%s' has invalid image name", name))
		}

		for _, runAfter := range goal.RunAfter {
			if _, ok := config.Goals[runAfter]; !ok {
				return errors.New(fmt.Sprintf("Goal '%s' should run after goal '%s' that does not exist", name, runAfter))
			}
		}

		// Goal 'test' links goal 'test2' that does not exist
		for _, linkedContainer := range goal.LinkedContainers() {
			if _, ok := config.Goals[linkedContainer.Name]; !ok {
				return errors.New(fmt.Sprintf("Goal '%s' links goal '%s' that does not exist", name, linkedContainer.Name))
			}
		}
	}
	return nil
}
