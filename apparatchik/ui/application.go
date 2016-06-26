package ui

import (
	"errors"
	"fmt"

	"github.com/netice9/apparatchik/apparatchik/core"
	"gitlab.netice9.com/dragan/go-bootreactor"
)

var goalRowUI = bootreactor.MustParseDisplayModel(`
  <tr id="row">
    <td ><a id="goal_name" href="#" className="btn btn-default"/></td>
    <td id="goal_state" />
    <td id="goal_actions" />
  </tr>
`)
var applicationUI = bootreactor.MustParseDisplayModel(`
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

      <bs.Button id="delete_app_button" bsStyle="danger" reportEvents="click">Delete!
        <bs.Modal id="delete_confirm_modal" bool:show="false" reportEvents="hide">
            <bs.Modal.Header>
              <bs.Modal.Title>Confirm Deleting Application</bs.Modal.Title>
            </bs.Modal.Header>

            <bs.Modal.Body>
              You are about to delete application "<strong id="application_name"/>". Are you sure?
            </bs.Modal.Body>

            <bs.Modal.Footer>
              <bs.Button id="delete_confirm_button" bsStyle="danger" reportEvents="click">Delete</bs.Button>
              <bs.Button id="cancel_delete_button" bsStyle="primary" reportEvents="click">Cancel</bs.Button>
            </bs.Modal.Footer>

        </bs.Modal>
      </bs.Button>

    </bs.Panel>
  </div>
`)

func Application(app *core.Application) func(*Context) (Screen, error) {

	applicationView := func(status *core.ApplicationStatus, showModal bool, alert error) *bootreactor.DisplayModel {
		view := applicationUI.DeepCopy()
		view.SetElementAttribute("app_panel", "header", status.Name)
		view.SetElementText("main_goal", app.MainGoal)

		for name, goal := range status.Goals {
			row := goalRowUI.DeepCopy()
			row.SetElementText("goal_name", name)
			row.SetElementAttribute("goal_name", "href", fmt.Sprintf("#/apps/%s/%s", app.Name, goal.Name))
			row.SetElementText("goal_state", goal.Status)
			view.AppendChild("goal_table_body", row)
		}

		if alert != nil {
			view.SetElementText("alert", alert.Error())
		} else {
			view.DeleteChild("alert")
		}

		view.SetElementAttribute("delete_confirm_modal", "show", showModal)
		view.SetElementText("application_name", app.Name)

		return WithNavigation(view, [][]string{{"Home", "#/"}, {app.Name, fmt.Sprintf("#/apps/%s", app.Name)}})
	}

	return func(ctx *Context) (Screen, error) {
		appUpdates := make(chan interface{})
		app.AddListener(appUpdates)
		showModal := false

		var applicationStatus *core.ApplicationStatus

		title := fmt.Sprintf("Apparatchik: Application %s", app.Name)

		ctx.display <- &bootreactor.DisplayUpdate{
			Title: &title,
		}

		var alert error

		for {
			select {
			case update, appActive := <-appUpdates:
				if !appActive {
					location := "#/"
					ctx.display <- &bootreactor.DisplayUpdate{
						Location: &location,
					}
				} else {
					applicationStatus = update.(*core.ApplicationStatus)
					ctx.display <- &bootreactor.DisplayUpdate{
						Model: applicationView(applicationStatus, showModal, alert),
					}
				}
			case evt, evtRead := <-ctx.userEvents:
				if !evtRead {
					return nil, errors.New("client disconnected")
				}
				if evt.ElementID == "delete_app_button" {
					showModal = true
					ctx.display <- &bootreactor.DisplayUpdate{
						Model: applicationView(applicationStatus, showModal, alert),
					}
				}
				if evt.ElementID == "delete_confirm_modal" && evt.Type == "hide" {
					showModal = false
					ctx.display <- &bootreactor.DisplayUpdate{
						Model: applicationView(applicationStatus, showModal, alert),
					}
				}
				if evt.ElementID == "cancel_delete_button" {
					showModal = false
					ctx.display <- &bootreactor.DisplayUpdate{
						Model: applicationView(applicationStatus, showModal, alert),
					}
				}
				if evt.ElementID == "delete_confirm_button" {
					showModal = false
					alert = errors.New("Deleting application.")
					ctx.display <- &bootreactor.DisplayUpdate{
						Model: applicationView(applicationStatus, showModal, alert),
					}
					_ = ctx.apparatchik.TerminateApplication(app.Name)
				}

				screen := ctx.ScreenForEvent(evt)
				if screen != nil {
					return screen, nil
				}
			}
		}
	}
}
