package ui

import (
	"errors"

	"github.com/netice9/apparatchik/apparatchik/core"
	"gitlab.netice9.com/dragan/go-bootreactor"
)

var goalRowUI = bootreactor.MustParseDisplayModel(`
  <tr id="row">
    <td id="goal_name" />
    <td id="goal_state" />
    <td id="goal_actions" />
  </tr>
`)
var applicationUI = bootreactor.MustParseDisplayModel(`
  <div>
    <bs.Panel id="app_panel" header="">
      <dl>
        <dt>Main Goal</dt>
        <dd id="main_goal"></dd>
      </dl>
      <dl>
        <dt>Goals</dt>
        <dd>
          <bs.Table bool:striped="true" bool:bordered="true" bool:condensed="true" bool:hover="true">
            <thead>
              <th>Name</th>
              <th>State</th>
              <th>Actions</th>
            </thead>
            <tbody id="goal_table_body" />

          </bs.Table>
        </dd>
      </dl>

    </bs.Panel>
  </div>
`)

func Application(app *core.Application) func(*Context) (Screen, error) {

	applicationView := func(status *core.ApplicationStatus) *bootreactor.DisplayModel {
		view := applicationUI.DeepCopy()
		view.SetElementAttribute("app_panel", "header", status.Name)
		view.SetElementText("main_goal", app.MainGoal)

		for name, goal := range status.Goals {
			row := goalRowUI.DeepCopy()
			row.SetElementText("goal_name", name)
			row.SetElementText("goal_state", goal.Status)
			view.AppendChild("goal_table_body", row)
		}

		return navigationUI.DeepCopy().ReplaceChild("content", view)
	}

	return func(ctx *Context) (Screen, error) {
		appUpdates := make(chan interface{})
		app.AddListener(appUpdates)

		for {
			select {
			case update, appActive := <-appUpdates:
				if !appActive {
					location := "#/"
					ctx.display <- &bootreactor.DisplayUpdate{
						Location: &location,
					}
				} else {
					ctx.display <- &bootreactor.DisplayUpdate{
						Model: applicationView(update.(*core.ApplicationStatus)),
					}
				}
			case evt, evtRead := <-ctx.userEvents:
				if !evtRead {
					return nil, errors.New("client disconnected")
				}
				screen := ctx.ScreenForEvent(evt)
				if screen != nil {
					return screen, nil
				}
			}
		}
	}
}
