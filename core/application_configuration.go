package core

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type ApplicationConfiguration struct {
	Goals    map[string]*GoalConfiguration `json:"goals"`
	MainGoal string                        `json:"main_goal"`
}

func (a *ApplicationConfiguration) findCircularDependency(goalName string, seen ...string) error {

	gc, found := a.Goals[goalName]

	if !found {
		return fmt.Errorf("Goal %q does not exist", goalName)
	}

	for _, d := range gc.dependsOn() {
		for _, s := range seen {
			if s == d {
				return fmt.Errorf("Goal %q has a circular dependency %q.", s, goalName)
			}
		}
	}

	seen = append(seen, gc.dependsOn()...)

	for _, d := range gc.dependsOn() {
		err := a.findCircularDependency(d, seen...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *ApplicationConfiguration) validateCircularDependencies() error {
	for g := range a.Goals {
		err := a.findCircularDependency(g, g)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO { "identitytoken": "9cbaf023786cd7..." }
type AuthConfiguration struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Email         string `json:"email,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`
}

func (a AuthConfiguration) toDockerAuthConfig() string {
	data, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

type GoalConfiguration struct {
	Image         string            `json:"image"`
	Command       []string          `json:"command,omitempty"`
	RunAfter      []string          `json:"run_after,omitempty"`
	Links         []string          `json:"links,omitempty"`
	ExtraHosts    []string          `json:"extra_hosts,omitempty"`
	Ports         []string          `json:"ports,omitempty"`
	Expose        []string          `json:"expose,omitempty"`
	Volumes       []string          `json:"volumes,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	LogDriver     string            `json:"log_driver,omitempty"`
	LogConfig     map[string]string `json:"log_config,omitempty"`
	Net           string            `json:"net,omitempty"`
	Dns           []string          `json:"dns,omitempty"`
	CapAdd        []string          `json:"cap_add,omitempty"`
	CapDrop       []string          `json:"cap_drop,omitempty"`
	DNSSearch     []string          `json:"dns_search,omitempty"`
	Devices       []string          `json:"devices,omitempty"`
	SecurityOpt   []string          `json:"security_opt,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty"`
	Entrypoint    []string          `json:"entrypoint,omitempty"`
	User          string            `json:"user,omitempty"`
	Hostname      string            `json:"hostname,omitempty"`
	Domainname    string            `json:"domainname,omitempty"`
	MacAddress    string            `json:"mac_address,omitempty"`
	MemLimit      int64             `json:"mem_limit,omitempty"`
	MemSwapLimit  int64             `json:"memswap_limit,omitempty"`
	Privileged    bool              `json:"privileged,omitempty"`
	Restart       string            `json:"restart,omitempty"`
	StdinOpen     bool              `json:"stdin_open,omitempty"`
	AttachStdin   bool              `json:"attach_stdin,omitempty"`
	AttachStdout  bool              `json:"attach_stdout,omitempty"`
	AttachStderr  bool              `json:"attach_stderr,omitempty"`
	Tty           bool              `json:"tty,omitempty"`
	CpuShares     int64             `json:"cpu_shares,omitempty"`
	CpuSet        string            `json:"cpuset,omitempty"`
	ReadOnly      bool              `json:"read_only,omitempty"`
	VolumeDriver  string            `json:"volume_driver,omitempty"`
	AuthConfig    AuthConfiguration `json:"auth_config,omitempty"`
	ContainerName string            `json:"container_name,omitempty"`
	ExternalLinks []string          `json:"external_links,omitempty"`
	SmartRestart  bool              `json:"smart_restart,omitempty"`
}

func (gc *GoalConfiguration) dependsOn() []string {
	depsMap := map[string]struct{}{}

	for _, lc := range gc.LinkedContainers() {
		depsMap[lc.Name] = struct{}{}
	}

	for _, ra := range gc.RunAfter {
		depsMap[ra] = struct{}{}
	}

	deps := []string{}

	for k := range depsMap {
		deps = append(deps, k)
	}

	sort.Strings(deps)

	return deps
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

func (c *ApplicationConfiguration) Clone() *ApplicationConfiguration {
	clone := *c
	clone.Goals = map[string]*GoalConfiguration{}
	for goalName, goal := range c.Goals {

		clone.Goals[goalName] = goal.Clone()
	}
	return &clone
}

var goalNameExpression = regexp.MustCompile("^[0-9a-zA-Z_\\.\\-]+$")

var imageExpression = regexp.MustCompile("^[0-9a-zA-Z\\.\\-/:_]+:[0-9a-zA-Z\\.\\-_]+$")

func (c *ApplicationConfiguration) Validate() error {
	if c.MainGoal == "" {
		return errors.New("Main goal is not set")
	}
	if _, ok := c.Goals[c.MainGoal]; !ok {
		return fmt.Errorf("Main goal %q is not defined", c.MainGoal)
	}
	for name, goal := range c.Goals {
		if !goalNameExpression.MatchString(name) {
			return fmt.Errorf("Goal %q has invalid name", name)
		}
		if !imageExpression.MatchString(goal.Image) {
			return fmt.Errorf("Goal %q has invalid image name", name)
		}

		for _, runAfter := range goal.RunAfter {
			if _, ok := c.Goals[runAfter]; !ok {
				return fmt.Errorf("Goal %q should run after goal %q that does not exist", name, runAfter)
			}
		}

		// Goal 'test' links goal 'test2' that does not exist
		for _, linkedContainer := range goal.LinkedContainers() {
			if _, ok := c.Goals[linkedContainer.Name]; !ok {
				return fmt.Errorf("Goal %q links goal %q that does not exist", name, linkedContainer.Name)
			}
		}
	}

	err := c.validateCircularDependencies()
	if err != nil {
		return err
	}

	return nil
}
