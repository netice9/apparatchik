package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/apparatchik/apparatchik/core"
	"github.com/netice9/apparatchik/apparatchik/util"
	"gitlab.netice9.com/dragan/go-bootreactor"
)

// 10 000 000

var goalUI = bootreactor.MustParseDisplayModel(`
	<div>
	  <bs.Panel id="goal_panel" header="CPU Stats">
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 130" width="100%" class="chart">
				<g transform="translate(10,20)">
					<path d="M28 0h3M28 100h3M31 100v3" strokeWidth="1px" stroke="#333"/>
					<path d="M31 0v100M31 100h400" strokeWidth="1px" stroke="#333"/>
					<polyline transform="translate(40,0)" id="cpu_line" fill="none" stroke="#0074d9" strokeWidth="1" points=""/>
					<g fontSize="8px" fontFamily="Georgia" fill="#333">
						<g textAnchor="end">
							<text id="max_cpu" x="26" y="2">100 %</text>
							<text x="26" y="102">0 %</text>
						</g>
					</g>
				</g>
			</svg>
		</bs.Panel>
		<bs.Panel id="goal_panel" header="Memory Stats">
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 130" width="100%" class="chart">
				<g transform="translate(10,20)">
				  <polyline transform="translate(40,0)" id="memory_line" fill="none" stroke="#0074d9" strokeWidth="1" points=""/>
					<path d="M28 0h3M28 100h3M31 100v3" strokeWidth="1px" stroke="#333"/>
					<path d="M31 0v100M31 100h400" strokeWidth="1px" stroke="#333"/>
					<g fontSize="8px" fontFamily="Georgia" fill="#333">
						<g textAnchor="end">
							<text id="max_memory" x="26" y="2">100 MB</text>
							<text x="26" y="102">0 MB</text>
						</g>
					</g>
				</g>
			</svg>
	  </bs.Panel>
		<bs.Panel id="output_panel" header="Output">
			<pre id="out" reportEvents="wheel:PD:X-deltaY" />
		</bs.Panel>
	</div>
`)

const outputHeight = 25

// Goal renders goal screen
func Goal(goal *core.Goal) func(*Context) (Screen, error) {
	goalView := func(stat core.GoalStatus, output []string, fromLine int) *bootreactor.DisplayModel {
		view := goalUI.DeepCopy()

		cpuPoints, cpuMax := util.TimeSeriesToLine(stat.Stats.CpuStats, 400, 100, 1000000)
		view.SetElementAttribute("cpu_line", "points", cpuPoints)
		cpuPercentMax := float64(cpuMax) / 10000000

		view.SetElementText("max_cpu", fmt.Sprintf("%.1f%%", cpuPercentMax))

		memoryPoints, memoryMax := util.TimeSeriesToLine(stat.Stats.MemStats, 400, 100, 1024*1024)
		memoryMBytes := memoryMax / (1024 * 1024)
		view.SetElementText("max_memory", fmt.Sprintf("%d MB", memoryMBytes))

		view.SetElementAttribute("memory_line", "points", memoryPoints)
		lastLine := fromLine + outputHeight
		if lastLine > len(output) {
			lastLine = len(output)
		}

		view.SetElementAttribute("output_panel", "header", fmt.Sprintf("Output: Lines %d - %d of %d", fromLine, lastLine, len(output)))

		view.SetElementText("out", strings.Join(output[fromLine:lastLine], "\n")+" ")

		return WithNavigation(view, [][]string{{"Applications", "#/"}, {goal.ApplicationName, fmt.Sprintf("#/apps/%s", goal.ApplicationName)}, {goal.Name, fmt.Sprintf("#/apps/%s/%s", goal.ApplicationName, goal.Name)}})
	}
	return func(ctx *Context) (Screen, error) {

		goalUpdates := goal.AddListener(1)
		defer goal.RemoveListener(goalUpdates)

		title := fmt.Sprintf("Apparatchik: Application %s Goal %s", goal.ApplicationName, goal.Name)

		ctx.display <- &bootreactor.DisplayUpdate{
			Title: &title,
		}

		containerID := goal.GetContainerID()

		tracker := util.NewOutputTracker(2000)

		if containerID != nil {
			go func() {
				err := goal.DockerClient.Logs(docker.LogsOptions{
					Container:    *containerID,
					OutputStream: tracker,
					ErrorStream:  tracker,
					Stdout:       true,
					Stderr:       true,
					Follow:       true,
					Tail:         "all",
					Timestamps:   true,
				})

				if err != nil {
					print(err)
				}

			}()
		}

		fromLine := 0

		outputTrackerUpdates := tracker.AddListener(0)

		output := []string{}
		stat := core.GoalStatus{}

		for {
			select {
			case output = <-outputTrackerUpdates:
				ctx.display <- &bootreactor.DisplayUpdate{
					Model: goalView(stat, output, fromLine),
				}
			case goalUpdate, updateReceived := <-goalUpdates:
				if !updateReceived {
					return MainScreen, nil
				}
				stat = goalUpdate
				ctx.display <- &bootreactor.DisplayUpdate{
					Model: goalView(stat, output, fromLine),
				}
			case evt, eventRead := <-ctx.userEvents:

				if eventRead {

					screeen := ctx.ScreenForEvent(evt)
					if screeen != nil {
						return screeen, nil
					}

					if evt.Type == "wheel" {
						deltaY := evt.ExtraValues["deltaY"].(float64)
						if deltaY > 0 {
							if (fromLine + outputHeight) < len(output) {
								fromLine++
							}
						} else {
							if fromLine > 0 {
								fromLine--
							}
						}
						ctx.display <- &bootreactor.DisplayUpdate{
							Model: goalView(stat, output, fromLine),
						}
					}
				} else {
					return nil, errors.New("client disconnected")
				}
			}
		}

		// return nil, nil

	}

}
