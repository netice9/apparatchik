package ui

import (
	"fmt"
	"sync"

	"github.com/draganm/go-reactor"
	"github.com/netice9/apparatchik/core"
	"github.com/netice9/apparatchik/util"
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
	ctx         reactor.ScreenContext
	goal        *core.Goal
	stat        core.GoalStatus
	fromLine    int
	containerID string
}

func (g *Goal) Mount() {
	g.goal.On("update", g.onGoalStatus)
	g.stat = g.goal.Status()
	g.render()

}

func (g *Goal) OnUserEvent(evt *reactor.UserEvent) {
}

func (g *Goal) render() {
	view := goalUI.DeepCopy()

	cpuPoints, cpuMax := util.TimeSeriesToLine(g.stat.Stats.CpuStats, 400, 100, 1000000)
	view.SetElementAttribute("cpu_line", "points", cpuPoints)
	cpuPercentMax := float64(cpuMax) / 10000000

	view.SetElementText("max_cpu", fmt.Sprintf("%.1f%%", cpuPercentMax))

	memoryPoints, memoryMax := util.TimeSeriesToLine(g.stat.Stats.MemStats, 400, 100, 1024*1024)
	memoryMBytes := memoryMax / (1024 * 1024)
	view.SetElementText("max_memory", fmt.Sprintf("%d MB", memoryMBytes))
	view.SetElementAttribute("memory_line", "points", memoryPoints)

	g.ctx.UpdateScreen(&reactor.DisplayUpdate{
		Model: WithNavigation(view, [][]string{{"Applications", "#/"}, {g.goal.ApplicationName, fmt.Sprintf("#/apps/%s", g.goal.ApplicationName)}, {g.goal.Name, fmt.Sprintf("#/apps/%s/%s", g.goal.ApplicationName, g.goal.Name)}}),
	})

}

func (g *Goal) Unmount() {
	g.goal.RemoveListener("update", g.onGoalStatus)
}

func (g *Goal) onGoalStatus(status core.GoalStatus) {
	g.Lock()
	defer g.Unlock()
	g.stat = status
	g.render()
}

var goalUI = reactor.MustParseDisplayModel(`
	<div>
	  <bs.Panel id="goal_panel" header="CPU Stats">
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 130" width="100%" className="chart">
				<g transform="translate(10,20)">
					<path d="M28 0h3M28 100h3M31 100v3" strokeWidth="1px" stroke="#333"/>
					<path d="M31 0v100M31 100h400" strokeWidth="1px" stroke="#333"/>
					<polyline transform="translate(32,0)" id="cpu_line" fill="none" stroke="#0074d9" strokeWidth="1" points=""/>
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
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 130" width="100%" className="chart">
				<g transform="translate(10,20)">
				  <polyline transform="translate(32,0)" id="memory_line" fill="none" stroke="#0074d9" strokeWidth="1" points=""/>
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
