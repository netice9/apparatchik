package dispatcher

import (
	"fmt"
	"reflect"
	"strings"
)

type Dispatcher struct {
	channelCases []reflect.SelectCase
	channelNames []string
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (d *Dispatcher) AddChannel(name string, channel interface{}) {
	d.channelCases = append(d.channelCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(channel),
	})
	d.channelNames = append(d.channelNames, name)
}

func (d *Dispatcher) deleteChannelCase(index int) {
	channelCasesCopy := make([]reflect.SelectCase, len(d.channelCases))
	channelNamesCopy := make([]string, len(d.channelNames))
	j := 0
	for i := range d.channelCases {
		if index != i {
			channelCasesCopy[j] = d.channelCases[i]
			channelNamesCopy[j] = d.channelNames[i]
		}
		j++
	}
	d.channelCases = channelCasesCopy
	d.channelNames = channelNamesCopy

}

func findValidMethodByName(values []reflect.Value, name string) reflect.Value {
	for _, v := range values {
		m := v.MethodByName(name)
		if m.IsValid() {
			return m
		}
	}
	return reflect.Value{}
}

func (d *Dispatcher) Dispatch(receivers ...interface{}) {

	values := []reflect.Value{}

	for _, receiver := range receivers {
		values = append(values, reflect.ValueOf(receiver))
	}

	defer func() {
		m := findValidMethodByName(values, "DispatcherFinished")
		if m.IsValid() {
			m.Call([]reflect.Value{})
		}
	}()

	for {
		selected, val, readOK := reflect.Select(d.channelCases)
		name := d.channelNames[selected]
		if !readOK {
			methodName := fmt.Sprintf("Closed%s%s", strings.ToUpper(name[:1]), name[1:])
			m := findValidMethodByName(values, methodName)
			if m.IsValid() {
				// TODO - catch panics!
				res := m.Call([]reflect.Value{})
				if len(res) != 1 {
					// TODO wtf?
					return
				}
				if res[0].Kind() != reflect.Bool {
					// TODO wtf?
					return
				}
				if res[0].Bool() {
					d.deleteChannelCase(selected)
					if len(d.channelCases) == 0 {
						return
					}
				} else {
					return
				}
			}
			return
		}
		methodName := fmt.Sprintf("Received%s%s", strings.ToUpper(name[:1]), name[1:])
		m := findValidMethodByName(values, methodName)

		if m.IsValid() {
			// TODO - catch panics!
			m.Call([]reflect.Value{val})
		} else {
			fmt.Println("not found method", methodName)
		}
	}

}
