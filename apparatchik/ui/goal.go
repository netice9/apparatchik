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

var goalUI = bootreactor.MustParseDisplayModel(`
  <bs.Panel id="goal_panel" header="">
		<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 200" width="100%" class="chart">
		  <polyline transform="translate(100,20)" id="cpu_line" fill="none" stroke="#0074d9" stroke-width="3" points=""/>
			<path d="M61 159v-3M93.714 159v-3M126.43 159v-3M159.14 159v-3M191.86 159v-3M224.57 159v-3M257.29 159v-3M290 159v-3M58 156h3M58 132.6h3M58 109.2h3M58 85.8h3M58 62.4h3M58 39h3" stroke-width="1px" stroke="#333"/>
			<path d="M58 156h235"/>
			<path d="M61 36v123"/>
			<g font-size="8px" font-family="Georgia" fill="#333">
				<g text-anchor="end">
					<text x="56" y="159">0</text>
					<text x="56" y="135.6">3.6</text>
					<text x="56" y="112.2">7.2</text>
					<text x="56" y="88.8">10.8</text>
					<text x="56" y="65.4">14.4</text>
					<text x="56" y="42">18</text>
				</g>
				<g text-anchor="middle">
					<text y="168" x="77.357">Mon</text>
					<text y="168" x="110.07">Tue</text>
					<text y="168" x="142.79">Wed</text>
					<text y="168" x="175.5">Thu</text>
					<text y="168" x="208.21">Fri</text>
					<text y="168" x="240.93">Sat</text>
					<text y="168" x="273.64">Sun</text>
				</g>
				<text text-anchor="middle" font-family="sans-serif" fill="#000" y="184" x="175.5">Days of the week</text>
				<text text-anchor="middle" font-family="sans-serif" fill="#000" y="97.5" x="26" transform="rotate(270,26,97.5)">Hours awake</text>
			</g>
		</svg>
		<svg viewBox="0 0 300 200" width="100%" class="chart">
		  <polyline id="memory_line" fill="none" stroke="#0074d9" stroke-width="3" points=""/>
		</svg>
  </bs.Panel>
	<bs.Panel header="Output">
		<pre id="out" reportEvents="wheel:PD:X-deltaY" />
	</bs.Panel>
  <span>TBD!</span>
`)

const outputHeight = 25

func Goal(goal *core.Goal) func(*Context) (Screen, error) {
	goalView := func(stat core.GoalStatus, output []string, fromLine int) *bootreactor.DisplayModel {
		view := goalUI.DeepCopy()

		view.SetElementAttribute("cpu_line", "points", util.TimeSeriesToLine(stat.Stats.CpuStats, 500, 100))
		view.SetElementAttribute("memory_line", "points", util.TimeSeriesToLine(stat.Stats.MemStats, 500, 100))

		lastLine := fromLine + outputHeight
		if lastLine > len(output) {
			lastLine = len(output)
		}

		view.SetElementText("out", strings.Join(output[fromLine:lastLine], "\n")+" ")

		// view.SetElementAttribute("app_panel", "header", status.Name)
		// view.SetElementText("main_goal", app.MainGoal)
		//
		// for name, goal := range status.Goals {
		//   row := goalRowUI.DeepCopy()
		//   row.SetElementText("goal_name", name)
		//   row.SetElementAttribute("goal_name", "href", fmt.Sprintf("#/apps/%s/%s", app.Name, goal.Name))
		//   row.SetElementText("goal_state", goal.Status)
		//   view.AppendChild("goal_table_body", row)
		// }
		//
		// if alert != nil {
		//   view.SetElementText("alert", alert.Error())
		// } else {
		//   view.DeleteChild("alert")
		// }
		//
		// view.SetElementAttribute("delete_confirm_modal", "show", showModal)
		// view.SetElementText("application_name", app.Name)
		//
		return WithNavigation(view, [][]string{{"Home", "#/"}, {goal.ApplicationName, fmt.Sprintf("#/apps/%s", goal.ApplicationName)}, {goal.Name, fmt.Sprintf("#/apps/%s/%s", goal.ApplicationName, goal.Name)}})
	}
	return func(ctx *Context) (Screen, error) {

		goalUpdates := goal.AddListener(0)
		defer goal.RemoveListener(goalUpdates)

		title := fmt.Sprintf("Apparatchik: Application %s Goal %s", goal.ApplicationName, goal.Name)

		ctx.display <- &bootreactor.DisplayUpdate{
			Title: &title,
		}

		containerId := goal.GetContainerID()

		tracker := util.NewOutputTracker(2000)

		go func() {
			goal.DockerClient.Logs(docker.LogsOptions{
				Container:    *containerId,
				OutputStream: tracker,
				ErrorStream:  tracker,
				Stdout:       true,
				Stderr:       true,
				Follow:       true,
				Tail:         "all",
				Timestamps:   true,
			})

		}()

		fromLine := 0

		outputTrackerUpdates := tracker.AddListener(0)
		defer close(outputTrackerUpdates)

		output := []string{}
		stat := core.GoalStatus{}

		for {
			select {
			case trackerUpdate := <-outputTrackerUpdates:

				output = trackerUpdate
				ctx.display <- &bootreactor.DisplayUpdate{
					Model: goalView(stat, output, fromLine),
				}

			case goalUpdate := <-goalUpdates:
				stat = goalUpdate
				ctx.display <- &bootreactor.DisplayUpdate{
					Model: goalView(stat, output, fromLine),
				}
				// fmt.Println(goalUpdate.Stats)
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
