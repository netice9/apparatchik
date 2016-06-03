package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/apparatchik/apparatchik/core"
	"github.com/netice9/cine"
)

// var apparatchick *core.Apparatchik = nil

func main() {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	cine.Init("localhost:8000")

	apparatchick, err := core.StartApparatchik(dockerClient)

	if err != nil {
		panic(err)
	}

	files, err := ioutil.ReadDir("/applications")

	if err != nil {
		panic(err)
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".json") {
			applicationName := name[0 : len(name)-len(".json")]
			data, err := ioutil.ReadFile("/applications/" + name)
			if err != nil {
				panic(err)
			}

			config := core.ApplicationConfiguration{}

			if err = json.Unmarshal(data, &config); err != nil {
				panic(err)
			}

			apparatchick.NewApplication(applicationName, &config)

		}

	}

	startHttpServer(apparatchick, dockerClient)
}
