package router

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/netice9/apparatchik/apparatchik/util/dispatcher"

	"gitlab.netice9.com/dragan/go-bootreactor/core"
)

type Resolver func(path string) Screen

type ScreenWrapper struct {
	Screen
}

func (s *ScreenWrapper) ReceivedUserEvent(evt *core.UserEvent) {
	val := reflect.ValueOf(s.Screen)
	elementID := evt.ElementID
	if len(elementID) > 1 {
		methodName := fmt.Sprintf("Evt%s%s", strings.ToUpper(elementID[:1]), elementID[1:])
		m := val.MethodByName(methodName)
		if m.IsValid() {
			m.Call([]reflect.Value{reflect.ValueOf(evt)})
		}
	}
}

func (s *ScreenWrapper) ClosedUserEvent() bool {
	s.Unmount()
	return false
}

type Screen interface {
	Mount(chan *core.DisplayUpdate) map[string]interface{}
	Unmount()
}

func NewRouterConnectionHandler(resolver Resolver) core.ClientConnectionHandler {
	return func(displayUpdate chan *core.DisplayUpdate, input chan *core.UserEvent, r *http.Request) http.Header {
		go func() {

			var dispatcherInput chan *core.UserEvent

			for evt := range input {
				if evt.ElementID == "main_window" && evt.Type == "popstate" {
					if dispatcherInput != nil {
						close(dispatcherInput)
						dispatcherInput = nil
					}
					newScreen := resolver(evt.Value)
					dispatcher := dispatcher.NewDispatcher()
					dispatcherInput = make(chan *core.UserEvent)
					dispatcher.AddChannel("userEvent", dispatcherInput)

					channels := newScreen.Mount(displayUpdate)

					if channels != nil {
						for name, channel := range channels {
							dispatcher.AddChannel(name, channel)
						}
					}

					wrapper := &ScreenWrapper{newScreen}

					go func() {
						dispatcher.Dispatch(wrapper, newScreen)
					}()
					// dispatcher.AddChannel("userEvent", input)
				} else {
					if dispatcherInput != nil {
						dispatcherInput <- evt
					}
				}
			}

		}()
		return http.Header{}
	}
}
