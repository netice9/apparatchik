package main

import (
	"log"
	"os"

	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/apparatchik/core"
	"gopkg.in/urfave/cli.v2"
)

// var apparatchick *core.Apparatchik = nil

func main() {

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "port",
				Value:       8080,
				DefaultText: "HTTP Port",
				EnvVars:     []string{"PORT"},
			},
		},
	}

	app.Action = func(ctx *cli.Context) error {
		dockerClient, err := docker.NewClientFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		apparatchick, err := core.StartApparatchik(dockerClient)

		if err != nil {
			panic(err)
		}

		core.ApparatchikInstance = apparatchick

		// files, err := ioutil.ReadDir("/applications")
		//
		// if err != nil {
		// 	panic(err)
		// }
		//
		// for _, file := range files {
		// 	name := file.Name()
		// 	if strings.HasSuffix(name, ".json") {
		// 		applicationName := name[0 : len(name)-len(".json")]
		// 		data, err := ioutil.ReadFile("/applications/" + name)
		// 		if err != nil {
		// 			panic(err)
		// 		}
		//
		// 		config := core.ApplicationConfiguration{}
		//
		// 		if err = json.Unmarshal(data, &config); err != nil {
		// 			panic(err)
		// 		}
		//
		// 		apparatchick.NewApplication(applicationName, &config)
		//
		// 	}
		//
		// }

		return startHttpServer(apparatchick, dockerClient, ctx.Int("port"))

	}

	app.Run(os.Args)

}
