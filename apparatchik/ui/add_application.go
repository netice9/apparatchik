package ui

import (
	"encoding/json"
	"strings"

	"github.com/netice9/apparatchik/apparatchik/core"

	bc "gitlab.netice9.com/dragan/go-bootreactor/core"
)

func addApplicationForm(alert error, hideFileForm bool, appName string) *bc.DisplayModel {
	addView := addApplicationUI.DeepCopy()

	addView.SetElementAttribute("app_name", "value", appName)

	if alert == nil {
		addView.DeleteChild("alert")
	} else {
		addView.SetElementText("alert", alert.Error())
	}

	if hideFileForm {
		addView.SetElementAttribute("descriptor", "disabled", true)
		addView.SetElementAttribute("deploy_btn", "disabled", false)
	} else {
		addView.DeleteChild("name_form")
		addView.DeleteChild("deploy_btn")
	}

	return WithNavigation(addView, [][]string{{"Applications", "#/"}, {"Add Application", "#/add_application"}})
}

var addApplicationUI = bc.MustParseDisplayModel(`
	<form>
		<bs.Alert id="alert" bsStyle="danger"/>
		<bs.FormGroup controlId="descriptorFile" id="file_form">
			<bs.ControlLabel>Application Descriptor File</bs.ControlLabel>
			<bs.FormControl id="descriptor" type="file" reportEvents="change"/>
			<bs.HelpBlock>File containing an application descriptor.</bs.HelpBlock>
		</bs.FormGroup>
		<bs.FormGroup controlId="descriptorFile" id="name_form">
			<bs.ControlLabel>Application Name</bs.ControlLabel>
			<bs.FormControl id="app_name" type="text" reportEvents="change"/>
			<bs.HelpBlock>File containing an application descriptor.</bs.HelpBlock>
		</bs.FormGroup>
		<bs.Button id="deploy_btn" reportEvents="click" bool:disabled="true">Deploy</bs.Button>
	</form>
`)

type AddApp struct {
	display     chan *bc.DisplayUpdate
	apparatchik *core.Apparatchik
	config      *core.ApplicationConfiguration
	appName     string
	alert       error
	location    *string
}

func (a *AddApp) render() {
	addView := addApplicationUI.DeepCopy()

	addView.SetElementAttribute("app_name", "value", a.appName)

	if a.alert == nil {
		addView.DeleteChild("alert")
	} else {
		addView.SetElementText("alert", a.alert.Error())
	}

	if a.config != nil {
		addView.SetElementAttribute("descriptor", "disabled", true)
		addView.SetElementAttribute("deploy_btn", "disabled", false)
	} else {
		addView.DeleteChild("name_form")
		addView.DeleteChild("deploy_btn")
	}

	view := WithNavigation(addView, [][]string{{"Applications", "#/"}, {"Add Application", "#/add_application"}})

	a.display <- &bc.DisplayUpdate{
		Model:    view,
		Location: a.location,
	}
}

func (a *AddApp) Mount(display chan *bc.DisplayUpdate) map[string]interface{} {
	a.display = display
	a.render()
	return nil
}

func (a *AddApp) Unmount() {

}

func (a *AddApp) EvtDescriptor(evt *bc.UserEvent) {
	config := &core.ApplicationConfiguration{}
	err := json.Unmarshal([]byte(evt.Data), &config)
	if err != nil {
		a.alert = err
		a.config = nil
	} else {
		a.alert = nil
		a.config = config
	}

	parts := strings.Split(evt.Value, ".")

	a.appName = parts[0]
	a.render()
}

func (a *AddApp) EvtApp_name(evt *bc.UserEvent) {
	a.appName = evt.Value
	a.render()
}

func (a *AddApp) EvtDeploy_btn(evt *bc.UserEvent) {
	_, err := a.apparatchik.NewApplication(a.appName, a.config)

	if err != nil {
		a.alert = err
	} else {
		newLoc := "#/"
		a.location = &newLoc
	}
	a.render()
}
