package ui

// import (
// 	"fmt"
//
// 	"github.com/netice9/apparatchik/apparatchik/core"
// 	bc "gitlab.netice9.com/dragan/go-bootreactor/core"
// )
//
// type XTerm struct {
// 	display chan *bc.DisplayUpdate
// 	goal    *core.Goal
// }
//
// var xtermView = bc.MustParseDisplayModel(`
// <bs.Panel header="Exec Terminal Session">
// 	<div id="container" data-api-path="/test" htmlID="terminal-container"></div>
// </bs.Panel>
// `)
//
// func (x *XTerm) render() {
// 	view := xtermView.DeepCopy()
//
// 	path := fmt.Sprintf("/api/v1.0/applications/%s/goals/%s/exec", x.goal.ApplicationName, x.goal.Name)
//
// 	view.SetElementAttribute("container", "data-api-path", path)
//
// 	x.display <- &bc.DisplayUpdate{
// 		Model: WithNavigation(view, [][]string{
// 			{"Applications", "#/"},
// 			{x.goal.ApplicationName, fmt.Sprintf("#/apps/%s", x.goal.ApplicationName)},
// 			{x.goal.Name, fmt.Sprintf("#/apps/%s/%s", x.goal.ApplicationName, x.goal.Name)},
// 			{"XTerm", fmt.Sprintf("#/apps/%s/%s/xterm", x.goal.ApplicationName, x.goal.Name)},
// 		}),
// 	}
//
// 	x.display <- &bc.DisplayUpdate{
// 		Eval: `
// 		startTerminal()
// 		`,
// 	}
// }
//
// func (x *XTerm) Mount(display chan *bc.DisplayUpdate) map[string]interface{} {
// 	x.display = display
// 	x.render()
// 	return nil
// }
//
// func (x *XTerm) Unmount() {
//
// }
