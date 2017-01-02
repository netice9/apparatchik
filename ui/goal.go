package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/draganm/go-reactor"
	"github.com/netice9/apparatchik/core"
	"github.com/netice9/apparatchik/core/stats"
)

func GoalFactory(ctx reactor.ScreenContext) reactor.Screen {

	applicationName := ctx.Params["application"]

	app, err := core.ApparatchikInstance.GetApplicationByName(applicationName)
	if err != nil {
		return reactor.DefaultNotFoundScreenFactory(ctx)
	}

	goalName := ctx.Params["goal"]

	goal, found := app.Goals[goalName]
	if !found {
		return reactor.DefaultNotFoundScreenFactory(ctx)
	}

	return &Goal{
		ctx:  ctx,
		goal: goal,
	}
}

type Goal struct {
	sync.Mutex
	ctx   reactor.ScreenContext
	goal  *core.Goal
	stat  core.GoalStatus
	stats []stats.Entry
	tail  string
}

func (g *Goal) Mount() {
	g.goal.On("update", g.onGoalStatus)
	g.goal.On("tail", g.onGoalTail)
	g.goal.On("stats", g.onGoalStats)
	g.stat = g.goal.Status()
	g.tail = g.goal.Tail()
	g.render()

}

func (g *Goal) OnUserEvent(evt *reactor.UserEvent) {
}

func (g *Goal) render() {
	view := renderGraph(g.stats)

	view.SetElementText("out", g.tail)

	g.ctx.UpdateScreen(&reactor.DisplayUpdate{
		Model: WithNavigation(view, [][]string{{"Applications", "#/"}, {g.goal.ApplicationName, fmt.Sprintf("#/apps/%s", g.goal.ApplicationName)}, {g.goal.Name, fmt.Sprintf("#/apps/%s/%s", g.goal.ApplicationName, g.goal.Name)}}),
	})

}

func (g *Goal) Unmount() {
	g.goal.RemoveListener("update", g.onGoalStatus)
	g.goal.RemoveListener("tail", g.onGoalTail)
	g.goal.RemoveListener("stats", g.onGoalStats)
}

func (g *Goal) onGoalStatus(status core.GoalStatus) {
	g.Lock()
	defer g.Unlock()
	g.stat = status
	g.render()
}

func (g *Goal) onGoalStats(stats []stats.Entry) {
	g.Lock()
	defer g.Unlock()
	g.stats = stats
	g.render()
}

func (g *Goal) onGoalTail(tail string) {
	g.Lock()
	defer g.Unlock()
	g.tail = tail
	g.render()
}

var goalUI = reactor.MustParseDisplayModel(`
	<div>
	  <bs.Panel id="goal_panel" header="CPU Stats">
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 130" width="100%" className="chart">
				<g transform="translate(10,20)">
					<path d="M28 0h3M28 100h3M31 100v3" strokeWidth="1px" stroke="#333"/>
					<path d="M31 0v100M31 100h400" strokeWidth="1px" stroke="#333"/>
					<polyline transform="translate(32,0)" id="cpuLine" fill="none" stroke="#0074d9" strokeWidth="1" points=""/>
					<g fontSize="8px" fontFamily="Georgia" fill="#333">
						<g textAnchor="end">
							<text id="maxCPU" x="26" y="2">100 %</text>
							<text x="26" y="102">0 %</text>
						</g>
					</g>
				</g>
			</svg>
		</bs.Panel>
		<bs.Panel id="goal_panel" header="Memory Stats">
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 130" width="100%" className="chart">
				<g transform="translate(10,20)">
				  <polyline transform="translate(32,0)" id="memLine" fill="none" stroke="#0074d9" strokeWidth="1" points=""/>
					<path d="M28 0h3M28 100h3M31 100v3" strokeWidth="1px" stroke="#333"/>
					<path d="M31 0v100M31 100h400" strokeWidth="1px" stroke="#333"/>
					<g fontSize="8px" fontFamily="Georgia" fill="#333">
						<g textAnchor="end">
							<text id="maxMem" x="26" y="2">100 MB</text>
							<text x="26" y="102">0 MB</text>
						</g>
					</g>
				</g>
			</svg>
	  </bs.Panel>
		<bs.Panel id="output_panel" header="Output">
			<pre id="out" className="pre-scrollable" />
		</bs.Panel>
	</div>
`)

type sample struct {
	time  time.Time
	value float64
}

func renderGraph(entries []stats.Entry) *reactor.DisplayModel {
	cpuSamples := []sample{}
	memSamples := []sample{}
	for _, e := range entries {
		cpuSamples = append(cpuSamples, sample{e.Time, float64(e.CPU) / 1e7})
		memSamples = append(memSamples, sample{e.Time, float64(e.Memory) / (1024 * 1024)})
	}

	g := goalUI.DeepCopy()

	cpuPoints, maxCPU := timeSeriesToLines(cpuSamples, 400, 100, 0.1)
	g.SetElementAttribute("cpuLine", "points", cpuPoints)
	g.SetElementText("maxCPU", fmt.Sprintf("%.1f%%", maxCPU))

	memPoints, maxMem := timeSeriesToLines(memSamples, 400, 100, 4.0)
	g.SetElementAttribute("memLine", "points", memPoints)
	g.SetElementText("maxMem", fmt.Sprintf("%.1f MB", maxMem))
	return g
}

func timeSeriesToLines(samples []sample, width, height int, lowestMax float64) (string, float64) {
	if len(samples) == 0 {
		return "", 0
	}

	minValue := float64(0)
	maxValue := lowestMax
	minTime := samples[0].time
	maxTime := samples[0].time

	for _, sample := range samples {
		if minValue > sample.value {
			minValue = sample.value
		}
		if maxValue < sample.value {
			maxValue = sample.value
		}
		if minTime.After(sample.time) {
			minTime = sample.time
		}
		if maxTime.Before(sample.time) {
			maxTime = sample.time
		}
	}

	points := []string{}

	valueRange := maxValue - minValue

	for _, sample := range samples {
		normalisedTime := float64(sample.time.UnixNano()-minTime.UnixNano()) / float64((time.Second * 120).Nanoseconds())

		scaledTime := int(normalisedTime * float64(width))

		normalisedValue := 1.0 - (sample.value / valueRange)
		scaledValue := int(normalisedValue * float64(height))
		points = append(points, fmt.Sprintf("%d,%d", scaledTime, scaledValue))
	}

	return strings.Join(points, " "), maxValue

}
