package ui

import (
	"fmt"

	"github.com/netice9/apparatchik/apparatchik/core"
	"gitlab.netice9.com/dragan/go-bootreactor"
)

var goalUI = bootreactor.MustParseDisplayModel(`
  <bs.Panel id="goal_panel" header="">

  </bs.Panel>
  <span>TBD!</span>
`)

func Goal(goal *core.Goal) func(*Context) (Screen, error) {
	goalView := func() *bootreactor.DisplayModel {
		view := goalUI.DeepCopy()
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
		return WithNavigation(view, [][]string{{"Home", "#/"}, {goal.Name, fmt.Sprintf("#/apps/%s", goal.ApplicationName)}, {goal.Name, fmt.Sprintf("#/apps/%s/%s", goal.ApplicationName, goal.Name)}})
	}
	return func(ctx *Context) (Screen, error) {

		_ = goal.AddListener(0)

		// var goalStatus *core.GoalStatus

		title := fmt.Sprintf("Apparatchik: Application %s Goal %s", goal.ApplicationName, goal.Name)

		ctx.display <- &bootreactor.DisplayUpdate{
			Model: goalView(),
			Title: &title,
		}

		for evt := range ctx.userEvents {
			screeen := ctx.ScreenForEvent(evt)
			if screeen != nil {
				return screeen, nil
			}
		}

		return nil, nil

	}

}
