package ui

import (
	"fmt"

	"github.com/draganm/go-reactor"
	"github.com/netice9/apparatchik/core"
)

var breadcrumbItemUI = reactor.MustParseDisplayModel(`
	<bs.Breadcrumb.Item id="breadcrumb_item" href="#" />
`)

var navigationUI = reactor.MustParseDisplayModel(`
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

func WithNavigation(content *reactor.DisplayModel, breadcrumbs [][]string) *reactor.DisplayModel {
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

var appGroupItem = reactor.MustParseDisplayModel(`<bs.ListGroupItem id="list_element" href="#link1">Link 1</bs.ListGroupItem>`)

var appGroupUI = reactor.MustParseDisplayModel(`
<div className="panel panel-default">
	<div className="panel-heading">
		<h3>Active Applications</h3>
	</div>
	<div className="panel-body">
		<bs.ListGroup id="list_group"/>
	</div>
	<div className="panel-footer">
		<bs.Button htmlID="deploy_button" href="#/add_application"><bs.Glyphicon glyph="plus"/> Deploy an Application</bs.Button>
	</div>
</div>
`)

func IndexFactory(ctx reactor.ScreenContext) reactor.Screen {
	return &Index{
		ctx: ctx,
	}
}

type Index struct {
	ctx          reactor.ScreenContext
	applications []string
}

func (i *Index) Mount() {
	core.ApparatchikInstance.AddListener("applications", i.onApplications)
	i.onApplications(core.ApparatchikInstance.ApplicatioNames())
}

func (i *Index) OnUserEvent(evt *reactor.UserEvent) {

}

func (i *Index) render() {
	listGroup := appGroupUI.DeepCopy()
	for _, app := range i.applications {
		item := appGroupItem.DeepCopy()
		item.SetElementText("list_element", app)
		item.SetElementAttribute("list_element", "href", fmt.Sprintf("#/apps/%s", app))
		listGroup.AppendChild("list_group", item)
	}

	i.ctx.UpdateScreen(&reactor.DisplayUpdate{
		Model: WithNavigation(listGroup, [][]string{{"Applications", "#/"}}),
	})

}

func (i *Index) onApplications(applications []string) {
	i.applications = applications
	i.render()
}

func (i *Index) Unmount() {
	core.ApparatchikInstance.RemoveListener("applications", i.onApplications)
}
