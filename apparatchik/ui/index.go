package ui

import (
	"fmt"
	"strings"

	"github.com/netice9/apparatchik/apparatchik/core"
	"github.com/netice9/apparatchik/apparatchik/util/router"
	bc "gitlab.netice9.com/dragan/go-bootreactor/core"
)

type Context struct {
	display     chan *bc.DisplayUpdate
	userEvents  chan *bc.UserEvent
	apparatchik *core.Apparatchik
}

func NewContext(display chan *bc.DisplayUpdate, userEvents chan *bc.UserEvent, apparatchik *core.Apparatchik) *Context {
	return &Context{
		display:     display,
		userEvents:  userEvents,
		apparatchik: apparatchik,
	}
}

var breadcrumbItemUI = bc.MustParseDisplayModel(`
	<bs.Breadcrumb.Item id="breadcrumb_item" href="#" />
`)

var navigationUI = bc.MustParseDisplayModel(`
  <div>
  	<bs.Navbar bool:fluid="true">
  		<bs.Navbar.Header>
  			<bs.Navbar.Brand>
  				<a href="#" className="navbar-brand">Apparatchik</a>
  			</bs.Navbar.Brand>
  		</bs.Navbar.Header>
  		<bs.Nav bool:pullRight="true">
  		 	<bs.NavItem href="#/add_application"><bs.Glyphicon glyph="plus"/></bs.NavItem>
  		 </bs.Nav>
  	</bs.Navbar>


  	<bs.Grid bool:fluid="true">
			<bs.Breadcrumb id="breadcrumb" />
			<bs.Row>
				<bs.Col int:mdOffset="1" int:md="10" int:smOffset="0" int:sm="12">

					<div id="content" className="container">Welcome!</div>
				</bs.Col>
			</bs.Row>
  	</bs.Grid>
  </div>
`)

func WithNavigation(content *bc.DisplayModel, breadcrumbs [][]string) *bc.DisplayModel {
	view := navigationUI.DeepCopy()
	if len(breadcrumbs) == 0 {
		view.DeleteChild("breadcrumb")
	} else {
		for i, item := range breadcrumbs {
			bcItem := breadcrumbItemUI.DeepCopy()
			bcItem.SetElementText("breadcrumb_item", item[0])
			bcItem.SetElementAttribute("breadcrumb_item", "href", item[1])
			if i == len(breadcrumbs)-1 {
				bcItem.SetElementAttribute("breadcrumb_item", "active", true)
			}
			view.AppendChild("breadcrumb", bcItem)
		}
	}
	view.ReplaceChild("content", content)
	return view
}

var appGroupItem = bc.MustParseDisplayModel(`<bs.ListGroupItem id="list_element" href="#link1">Link 1</bs.ListGroupItem>`)

var appGroupUI = bc.MustParseDisplayModel(`
<div className="panel panel-default">
	<div className="panel-heading">
		<h3>Active Applications</h3>
	</div>
	<div className="panel-body">
		<bs.ListGroup id="list_group"/>
	</div>
	<div className="panel-footer">
		<bs.Button draggable="true" href="#/add_application" reportEvents="contextMenu:PD:SP mouseUp:SP:PD:X-button:X-buttons mouseDown:SP:PD:X-button:X-buttons dragStart drag:X-pageX:X-pageY dragOver drop wheel:PD:X-deltaY"><bs.Glyphicon glyph="plus"/> Deploy an Application</bs.Button>
	</div>
</div>
`)

type MainS struct {
	apparatchik *core.Apparatchik
	display     chan *bc.DisplayUpdate
	apps        []string
	listener    chan []string
}

func (m *MainS) render() {
	listGroup := appGroupUI.DeepCopy()
	for _, app := range m.apps {
		item := appGroupItem.DeepCopy()
		item.SetElementText("list_element", app)
		item.SetElementAttribute("list_element", "href", fmt.Sprintf("#/apps/%s", app))
		listGroup.AppendChild("list_group", item)
	}

	m.display <- &bc.DisplayUpdate{
		Model: WithNavigation(listGroup, [][]string{{"Applications", "#/"}}),
	}

}

func (m *MainS) Mount(display chan *bc.DisplayUpdate) map[string]interface{} {
	m.listener = m.apparatchik.AddListener(0)
	m.display = display
	m.render()
	return map[string]interface{}{
		"apparatchik": m.listener,
	}
}

func (m *MainS) ReceivedApparatchik(apps []string) {
	fmt.Println(apps)
	m.apps = apps
	m.render()
}

func (m *MainS) Unmount() {
	m.apparatchik.RemoveListener(m.listener)
}

func PathResolver(apparatchik *core.Apparatchik) func(path string) router.Screen {
	return func(path string) router.Screen {

		fmt.Printf("path: %s\n", path)

		if path == "#/add_application" {
			return &AddApp{
				apparatchik: apparatchik,
			}
		}

		if strings.HasPrefix(path, "#/apps/") {

			parts := strings.Split(strings.TrimPrefix(path, "#/apps/"), "/")

			appName := parts[0]

			app, err := apparatchik.GetApplicationByName(appName)
			if err != nil {
				return &MainS{apparatchik: apparatchik}
			}

			if len(parts) == 2 {
				goal, found := app.Goals[parts[1]]
				if !found {
					return &AppS{app: app, apparatchik: apparatchik}
				}
				return &GoalS{
					goal: goal,
				}
			}
			return &AppS{app: app, apparatchik: apparatchik}

		}

		return &MainS{apparatchik: apparatchik}
	}
}
