package ui

import (
	"encoding/json"
	"strings"

	"github.com/netice9/apparatchik/apparatchik/core"

	"gitlab.netice9.com/dragan/go-bootreactor"
)

func addApplicationForm(alert error, hideFileForm bool, appName string) *bootreactor.DisplayModel {
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

	return navigationUI.DeepCopy().ReplaceChild("content", addView)
}

var addApplicationUI = bootreactor.MustParseDisplayModel(`
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

func AddApplicationScreen(ctx *Context) (Screen, error) {

	var config *core.ApplicationConfiguration
	appName := ""

	var alert error

	ctx.display <- &bootreactor.DisplayUpdate{
		Model: addApplicationForm(alert, config != nil, appName),
	}

	for evt := range ctx.userEvents {

		if evt.ElementID == "app_name" {
			appName = evt.Value
			ctx.display <- &bootreactor.DisplayUpdate{
				Model: addApplicationForm(alert, config != nil, appName),
			}
		}

		if evt.ElementID == "descriptor" && evt.Data != "" && evt.Value != "" {
			config = &core.ApplicationConfiguration{}
			err := json.Unmarshal([]byte(evt.Data), &config)
			if err != nil {
				alert = err
				config = nil
			} else {
				alert = nil
			}

			parts := strings.Split(evt.Value, ".")

			appName = parts[0]
			ctx.display <- &bootreactor.DisplayUpdate{
				Model: addApplicationForm(alert, config != nil, appName),
			}

		}

		if evt.ElementID == "deploy_btn" {
			_, err := ctx.apparatchik.NewApplication(appName, config)

			if err != nil {
				alert = err
			} else {
				location := "#/"
				ctx.display <- &bootreactor.DisplayUpdate{
					Location: &location,
				}
			}

			ctx.display <- &bootreactor.DisplayUpdate{
				Model: addApplicationForm(alert, config != nil, appName),
			}

		}

		next := ctx.ScreenForEvent(evt)
		if next != nil {
			return next, nil
		}
	}

	return MainScreen, nil
}
