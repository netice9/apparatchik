package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/netice9/apparatchik/apparatchik/core"
	"gitlab.netice9.com/dragan/go-bootreactor"
)

type Context struct {
	display     chan *bootreactor.DisplayUpdate
	userEvents  chan *bootreactor.UserEvent
	apparatchik *core.Apparatchik
}

type Screen func(*Context) (Screen, error)

func (c *Context) ScreenForEvent(evt *bootreactor.UserEvent) Screen {
	if evt.ElementID == "main_window" && evt.Type == "popstate" {
		if evt.Value == "#/add_application" {
			return AddApplicationScreen
		}
		if evt.Value == "#/" || evt.Value == "#" || evt.Value == "" {
			return MainScreen
		}
		if strings.HasPrefix(evt.Value, "#/apps/") {
			appName := strings.TrimPrefix(evt.Value, "#/apps/")
			app, err := c.apparatchik.GetApplicationByName(appName)
			if err != nil {
				return nil
			} else {
				return Application(app)
			}
		}
	}
	return nil
}

func NewContext(display chan *bootreactor.DisplayUpdate, userEvents chan *bootreactor.UserEvent, apparatchik *core.Apparatchik) *Context {
	return &Context{
		display:     display,
		userEvents:  userEvents,
		apparatchik: apparatchik,
	}
}

var navigationUI = bootreactor.MustParseDisplayModel(`
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
  	 <bs.Row>
  		 <bs.Col int:mdOffset="2" int:md="8" int:smOffset="0" int:sm="12">
			 	<div id="content" className="container">Welcome!</div>
			 </bs.Col>
  	 </bs.Row>
  	</bs.Grid>
  </div>
`)

var appGroupItem = bootreactor.MustParseDisplayModel(`<bs.ListGroupItem id="list_element" href="#link1">Link 1</bs.ListGroupItem>`)

var appGroupUI = bootreactor.MustParseDisplayModel(`
<div className="panel panel-default">
	<div className="panel-heading">
		<h3>Active Applications</h3>
	</div>
	<div className="panel-body">
		<bs.ListGroup id="list_group"/>
	</div>
	<div className="panel-footer">
		<bs.Button href="#/add_application"><bs.Glyphicon glyph="plus"/></bs.Button>
	</div>
</div>
`)

func MainScreen(ctx *Context) (Screen, error) {

	ch := make(chan interface{})

	ctx.apparatchik.AddListener(ch)
	defer ctx.apparatchik.RemoveListener(ch)

	for {
		select {
		case apps := <-ch:
			view := navigationUI.DeepCopy()
			listGroup := appGroupUI.DeepCopy()
			for _, app := range apps.([]string) {
				item := appGroupItem.DeepCopy()
				item.SetElementText("list_element", app)
				item.SetElementAttribute("list_element", "href", fmt.Sprintf("#/apps/%s", app))
				listGroup.AppendChild("list_group", item)
			}

			view.ReplaceChild("content", listGroup)
			ctx.display <- &bootreactor.DisplayUpdate{
				Model: view,
			}
		case evt, eventOK := <-ctx.userEvents:
			if !eventOK {
				return nil, errors.New("closed")
			}
			next := ctx.ScreenForEvent(evt)
			if next != nil {
				return next, nil
			}
		}
	}

}

func RunApparatchikUI(ctx *Context) {
	var err error
	var screen Screen = MainScreen
	for {
		screen, err = screen(ctx)
		if err != nil {
			close(ctx.display)
			return
		}
	}
}
