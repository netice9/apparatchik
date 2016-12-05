package reactor

import (
	"gitlab.netice9.com/dragan/go-reactor/core"
)

type DefaultNotFoundScreen struct {
	ctx ScreenContext
}

var defaultNotFoundScreenUI = core.MustParseDisplayModel(`
  <bs.PageHeader>Not Found <small>something went wrong</small></bs.PageHeader>
`)

func (d *DefaultNotFoundScreen) Mount() {
	d.ctx.UpdateScreen(&core.DisplayUpdate{Model: defaultNotFoundScreenUI})
}

func (d *DefaultNotFoundScreen) OnUserEvent(*core.UserEvent) {

}

func (d *DefaultNotFoundScreen) Unmount() {
}

func DefaultNotFoundScreenFactory(ctx ScreenContext) Screen {
	return &DefaultNotFoundScreen{ctx}
}
