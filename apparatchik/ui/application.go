package ui

import (
	"errors"
	"fmt"

	"github.com/netice9/apparatchik/apparatchik/core"
	bc "gitlab.netice9.com/dragan/go-bootreactor/core"
)

var goalRowUI = bc.MustParseDisplayModel(`
  <tr id="row">
    <td ><a id="goal_name" href="#" className="btn btn-default"/></td>
    <td id="goal_state" />
    <td id="goal_actions">
			<a id="goal_term_link" href="#" className="btn btn-default">XTerm</a>
		</td>
  </tr>
`)
var applicationUI = bc.MustParseDisplayModel(`
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

type AppS struct {
	display     chan *bc.DisplayUpdate
	updates     chan core.ApplicationStatus
	apparatchik *core.Apparatchik
	app         *core.Application
	alert       error
	status      core.ApplicationStatus
	showModal   bool
}

func (a *AppS) render() {
	view := applicationUI.DeepCopy()
	view.SetElementAttribute("app_panel", "header", a.status.Name)
	view.SetElementText("main_goal", a.app.MainGoal)

	for name, goal := range a.status.Goals {
		row := goalRowUI.DeepCopy()
		row.SetElementText("goal_name", name)
		row.SetElementAttribute("goal_name", "href", fmt.Sprintf("#/apps/%s/%s", a.app.Name, goal.Name))
		row.SetElementAttribute("goal_term_link", "href", fmt.Sprintf("#/apps/%s/%s/xterm", a.app.Name, goal.Name))
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

	a.display <- &bc.DisplayUpdate{
		Model: WithNavigation(view, [][]string{{"Applications", "#/"}, {a.app.Name, fmt.Sprintf("#/apps/%s", a.app.Name)}}),
	}
}

func (a *AppS) Mount(display chan *bc.DisplayUpdate) map[string]interface{} {
	a.display = display
	a.updates = a.app.AddListener(0)
	a.render()
	return map[string]interface{}{"applicationStatus": a.updates}
}

func (a *AppS) ReceivedApplicationStatus(status core.ApplicationStatus) {
	a.status = status
	a.render()
}

func (a *AppS) ClosedApplicationStatus() bool {
	location := "#/"
	a.display <- &bc.DisplayUpdate{
		Location: &location,
	}
	return false
}

func (a *AppS) EvtDeleteButton(evt *bc.UserEvent) {
	a.showModal = true
	a.render()
}

func (a *AppS) EvtDeleteCancelButton(evt *bc.UserEvent) {
	a.showModal = false
	a.render()
}

func (a *AppS) EvtDeleteConfirmButton(evt *bc.UserEvent) {
	a.showModal = false
	a.alert = errors.New("Deleting application.")

	err := a.apparatchik.TerminateApplication(a.app.Name)
	if err != nil {
		a.alert = err
	}
	a.render()
}

func (a *AppS) Unmount() {
	a.app.RemoveListener(a.updates)
}
