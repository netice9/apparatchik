package main

import (
	"errors"

	"github.com/fsouza/go-dockerclient"
)

type ApplicationConfiguration struct {
	Goals    map[string]GoalConfiguration `json:"goals"`
	MainGoal string                       `json:"main_goal"`
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

func (config *ApplicationConfiguration) Validate() error {
	if config.MainGoal == "" {
		return errors.New("Main goal is not set")
	}
	return nil
}
