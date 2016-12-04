package ui

// import (
// 	"fmt"
// 	"strings"
//
// 	"github.com/fsouza/go-dockerclient"
// 	"github.com/netice9/apparatchik/apparatchik/core"
// 	"github.com/netice9/apparatchik/apparatchik/util"
// 	bc "gitlab.netice9.com/dragan/go-bootreactor/core"
// )
//
// // 10 000 000
//
// var goalUI = bc.MustParseDisplayModel(`
// 	<div>
// 	  <bs.Panel id="goal_panel" header="CPU Stats">
// 			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 130" width="100%" class="chart">
// 				<g transform="translate(10,20)">
// 					<path d="M28 0h3M28 100h3M31 100v3" strokeWidth="1px" stroke="#333"/>
// 					<path d="M31 0v100M31 100h400" strokeWidth="1px" stroke="#333"/>
// 					<polyline transform="translate(40,0)" id="cpu_line" fill="none" stroke="#0074d9" strokeWidth="1" points=""/>
// 					<g fontSize="8px" fontFamily="Georgia" fill="#333">
// 						<g textAnchor="end">
// 							<text id="max_cpu" x="26" y="2">100 %</text>
// 							<text x="26" y="102">0 %</text>
// 						</g>
// 					</g>
// 				</g>
// 			</svg>
// 		</bs.Panel>
// 		<bs.Panel id="goal_panel" header="Memory Stats">
// 			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 130" width="100%" class="chart">
// 				<g transform="translate(10,20)">
// 				  <polyline transform="translate(40,0)" id="memory_line" fill="none" stroke="#0074d9" strokeWidth="1" points=""/>
// 					<path d="M28 0h3M28 100h3M31 100v3" strokeWidth="1px" stroke="#333"/>
// 					<path d="M31 0v100M31 100h400" strokeWidth="1px" stroke="#333"/>
// 					<g fontSize="8px" fontFamily="Georgia" fill="#333">
// 						<g textAnchor="end">
// 							<text id="max_memory" x="26" y="2">100 MB</text>
// 							<text x="26" y="102">0 MB</text>
// 						</g>
// 					</g>
// 				</g>
// 			</svg>
// 	  </bs.Panel>
// 		<bs.Panel id="output_panel" header="Output">
// 			<pre id="out" reportEvents="wheel:PD:X-deltaY" />
// 		</bs.Panel>
// 	</div>
// `)
//
// const outputHeight = 25
//
// type GoalS struct {
// 	display         chan *bc.DisplayUpdate
// 	goalUpdates     chan core.GoalStatus
// 	goal            *core.Goal
// 	stat            core.GoalStatus
// 	fromLine        int
// 	output          []string
// 	outputTracker   *util.OutputTracker
// 	containerID     string
// 	trackerListener chan []string
// }
//
// func (g *GoalS) render() {
// 	view := goalUI.DeepCopy()
//
// 	cpuPoints, cpuMax := util.TimeSeriesToLine(g.stat.Stats.CpuStats, 400, 100, 1000000)
// 	view.SetElementAttribute("cpu_line", "points", cpuPoints)
// 	cpuPercentMax := float64(cpuMax) / 10000000
//
// 	view.SetElementText("max_cpu", fmt.Sprintf("%.1f%%", cpuPercentMax))
//
// 	memoryPoints, memoryMax := util.TimeSeriesToLine(g.stat.Stats.MemStats, 400, 100, 1024*1024)
// 	memoryMBytes := memoryMax / (1024 * 1024)
// 	view.SetElementText("max_memory", fmt.Sprintf("%d MB", memoryMBytes))
//
// 	view.SetElementAttribute("memory_line", "points", memoryPoints)
// 	lastLine := g.fromLine + outputHeight
// 	if lastLine > len(g.output) {
// 		lastLine = len(g.output)
// 	}
//
// 	view.SetElementAttribute("output_panel", "header", fmt.Sprintf("Output: Lines %d - %d of %d", g.fromLine, lastLine, len(g.output)))
//
// 	view.SetElementText("out", strings.Join(g.output[g.fromLine:lastLine], "\n")+" ")
//
// 	g.display <- &bc.DisplayUpdate{
// 		Model: WithNavigation(view, [][]string{{"Applications", "#/"}, {g.goal.ApplicationName, fmt.Sprintf("#/apps/%s", g.goal.ApplicationName)}, {g.goal.Name, fmt.Sprintf("#/apps/%s/%s", g.goal.ApplicationName, g.goal.Name)}}),
// 	}
//
// }
//
// func (g *GoalS) Mount(display chan *bc.DisplayUpdate) map[string]interface{} {
// 	g.display = display
// 	g.goalUpdates = g.goal.AddListener(1)
//
// 	g.outputTracker = util.NewOutputTracker(2000)
//
// 	g.trackerListener = g.outputTracker.AddListener(1)
//
// 	g.render()
// 	return map[string]interface{}{
// 		"goalUpdate": g.goalUpdates,
// 		"output":     g.trackerListener,
// 	}
// }
//
// func (g *GoalS) Unmount() {
// 	g.goal.RemoveListener(g.goalUpdates)
// 	g.outputTracker.RemoveListener(g.trackerListener)
// 	g.outputTracker.Close()
// }
//
// func (g *GoalS) ReceivedOutput(output []string) {
// 	g.output = output
// 	g.render()
// }
//
// func (g *GoalS) EvtOut(evt *bc.UserEvent) {
// 	if evt.Type == "wheel" {
// 		deltaY := evt.ExtraValues["deltaY"].(float64)
// 		if deltaY > 0 {
// 			if (g.fromLine + outputHeight) < len(g.output) {
// 				g.fromLine++
// 			}
// 		} else {
// 			if g.fromLine > 0 {
// 				g.fromLine--
// 			}
// 		}
// 		g.render()
// 	}
// }
//
// func (g *GoalS) ReceivedGoalUpdate(status core.GoalStatus) {
// 	g.stat = status
//
// 	if g.containerID == "" && g.goal.ContainerId != nil {
// 		g.containerID = *g.goal.ContainerId
// 		go func() {
// 			err := g.goal.DockerClient.Logs(docker.LogsOptions{
// 				Container:    g.containerID,
// 				OutputStream: g.outputTracker,
// 				ErrorStream:  g.outputTracker,
// 				Stdout:       true,
// 				Stderr:       true,
// 				Follow:       true,
// 				Tail:         "all",
// 				Timestamps:   true,
// 			})
//
// 			if err != nil {
// 				print(err)
// 			}
//
// 		}()
// 	}
//
// 	g.render()
// }
