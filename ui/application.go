package ui

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/draganm/go-reactor"
	"github.com/netice9/apparatchik/core"
)

type Application struct {
	sync.Mutex
	ctx       reactor.ScreenContext
	app       *core.Application
	alert     error
	status    core.ApplicationStatus
	showModal bool
}

func ApplicationFactory(ctx reactor.ScreenContext) reactor.Screen {
	appName := ctx.Params["application"]

	app, err := core.ApparatchikInstance.GetApplicationByName(appName)

	if err != nil {
		return reactor.DefaultNotFoundScreenFactory(ctx)
	}

	return &Application{
		ctx: ctx,
		app: app,
	}
}

func (a *Application) Mount() {
	a.app.On("update", a.onUpdate)
	a.app.On("terminated", a.onTerminated)
	a.onUpdate(a.app.Status())
}

func (a *Application) OnUserEvent(evt *reactor.UserEvent) {
	a.Lock()
	defer a.Unlock()

	switch evt.ElementID {
	case "deleteButton":
		a.showModal = true
	case "deleteCancelButton":
		a.showModal = false
	case "deleteConfirmButton":
		a.showModal = false
		a.alert = errors.New("Deleting application.")

		err := core.ApparatchikInstance.TerminateApplication(a.app.Name)
		if err != nil {
			a.alert = err
		}

	}
	a.render()

}

func (a *Application) onUpdate(status core.ApplicationStatus) {
	a.Lock()
	defer a.Unlock()
	a.status = status
	a.render()
}

func (a *Application) onTerminated() {
	a.ctx.UpdateScreen(&reactor.DisplayUpdate{
		Location: "#/",
	})
}

func (a *Application) render() {
	view := applicationUI.DeepCopy()
	view.SetElementAttribute("app_panel", "header", a.status.Name)
	view.SetElementText("main_goal", a.app.MainGoal)

	goalNames := []string{}

	for goalName := range a.status.Goals {
		goalNames = append(goalNames, goalName)
	}

	sort.Strings(goalNames)

	for _, name := range goalNames {
		goal := a.status.Goals[name]
		row := goalRowUI.DeepCopy()
		row.SetElementText("goal_name", name)
		row.SetElementAttribute("goal_name", "href", fmt.Sprintf("#/apps/%s/%s", a.app.Name, goal.Name))
		if goal.Status == "running" {
			row.SetElementAttribute("goal_term_link", "href", fmt.Sprintf("#/apps/%s/%s/xterm", a.app.Name, goal.Name))
		} else {
			row.SetElementAttribute("goal_term_link", "disabled", true)
		}
		row.SetElementText("goal_state", goal.Status)
		view.AppendChild("goal_table_body", row)
	}

	if a.alert != nil {
		view.SetElementText("alert", a.alert.Error())
	} else {
		view.DeleteChild("alert")
	}

	view.SetElementAttribute("delete_confirm_modal", "show", a.showModal)
	view.SetElementText("application_name", a.app.Name)

	a.ctx.UpdateScreen(&reactor.DisplayUpdate{
		Model: WithNavigation(view, [][]string{{"Applications", "#/"}, {a.app.Name, fmt.Sprintf("#/apps/%s", a.app.Name)}}),
	})
}

func (a *Application) Unmount() {
	a.app.RemoveListener("update", a.onUpdate)
	a.app.RemoveListener("terminated", a.onTerminated)
}

var goalRowUI = reactor.MustParseDisplayModel(`
  <tr id="row">
    <td ><a id="goal_name" href="#" className="btn btn-default"/></td>
    <td id="goal_state" />
    <td id="goal_actions">
			<bs.ButtonToolbar>
				<bs.Button id="goal_term_link">XTerm</bs.Button>
			</bs.ButtonToolbar>
		</td>
  </tr>
`)
var applicationUI = reactor.MustParseDisplayModel(`
  <div>
    <bs.Panel id="app_panel" header="">
      <bs.Alert id="alert" bsStyle="danger"/>
      <dl>
        <dt>Main Goal</dt>
        <dd id="main_goal"></dd>
      </dl>
      <dl>
        <dt>Goals</dt>
        <dd>
          <bs.Table bool:striped="true" bool:bordered="true" bool:condensed="true" bool:hover="true">
            <thead>
              <tr>
                <th>Name</th>
                <th>State</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody id="goal_table_body" />
          </bs.Table>
        </dd>
      </dl>

      <bs.Button id="deleteButton" bsStyle="danger" reportEvents="click">Delete!</bs.Button>
			<bs.Modal id="delete_confirm_modal" bool:show="false" reportEvents="hide">
					<bs.Modal.Header>
						<bs.Modal.Title>Confirm Deleting Application</bs.Modal.Title>
					</bs.Modal.Header>

					<bs.Modal.Body>
						You are about to delete application "<strong id="application_name"/>". Are you sure?
					</bs.Modal.Body>

					<bs.Modal.Footer>
						<bs.Button id="deleteConfirmButton" bsStyle="danger" reportEvents="click">Delete</bs.Button>
						<bs.Button id="deleteCancelButton" bsStyle="primary" reportEvents="click">Cancel</bs.Button>
					</bs.Modal.Footer>
			</bs.Modal>

    </bs.Panel>
  </div>
`)
