package ui

import (
	"encoding/json"
	"strings"

	"github.com/draganm/go-reactor"
	"github.com/netice9/apparatchik/core"
)

type AddApplication struct {
	ctx      reactor.ScreenContext
	config   *core.ApplicationConfiguration
	appName  string
	alert    error
	location *string
}

func AddApplicationFactory(ctx reactor.ScreenContext) reactor.Screen {
	return &AddApplication{
		ctx: ctx,
	}
}

func (aa *AddApplication) Mount() {
	aa.render()
}

func (aa *AddApplication) OnUserEvent(evt *reactor.UserEvent) {
	switch evt.ElementID {
	case "descriptor":
		config := &core.ApplicationConfiguration{}
		err := json.Unmarshal([]byte(evt.Data), &config)
		if err != nil {
			aa.alert = err
			aa.config = nil
		} else {
			aa.alert = nil
			aa.config = config
		}

		parts := strings.Split(evt.Value, ".")

		aa.appName = parts[0]
		aa.render()
	case "app_name":
		aa.appName = evt.Value
		aa.render()
	case "deploy_btn":
		_, err := core.ApparatchikInstance.NewApplication(aa.appName, aa.config)

		if err != nil {
			aa.alert = err
		} else {
			newLoc := "#/"
			aa.location = &newLoc
		}
		aa.render()
	}
}

func (aa *AddApplication) render() {
	addView := addApplicationUI.DeepCopy()

	addView.SetElementAttribute("app_name", "value", aa.appName)

	if aa.alert == nil {
		addView.DeleteChild("alert")
	} else {
		addView.SetElementText("alert", aa.alert.Error())
	}

	if aa.config != nil {
		addView.SetElementAttribute("descriptor", "disabled", true)
		addView.SetElementAttribute("deploy_btn", "disabled", false)
	} else {
		addView.DeleteChild("name_form")
		addView.DeleteChild("deploy_btn")
	}

	view := WithNavigation(addView, [][]string{{"Applications", "#/"}, {"Add Application", "#/add_application"}})

	aa.ctx.UpdateScreen(&reactor.DisplayUpdate{
		Model:    view,
		Location: aa.location,
	})

}

func (aa *AddApplication) Unmount() {}

func addApplicationForm(alert error, hideFileForm bool, appName string) *reactor.DisplayModel {
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

var addApplicationUI = reactor.MustParseDisplayModel(`
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
