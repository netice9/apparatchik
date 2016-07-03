/*
* CODE GENERATED AUTOMATICALLY WITH github.com/ernesto-jimenez/gogen/specific
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package core

import "sync"

type ApparatchikNotifier struct {
	sync.Mutex
	listeners		[]chan []string
	lastNotification	[]string
}

func NewApparatchikNotifier(firstNotification []string) *ApparatchikNotifier {
	return &ApparatchikNotifier{
		lastNotification: firstNotification,
	}
}

func (n *ApparatchikNotifier) Notify(value []string) {
	n.Lock()
	defer n.Unlock()

	n.lastNotification = value
	for _, listener := range n.listeners {
		l := listener
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// ignore?
				}
			}()
			l <- value
		}()
	}

}

func (n *ApparatchikNotifier) AddListener(capacity int) chan []string {
	listenerChannel := make(chan []string, capacity)
	n.Lock()
	defer n.Unlock()
	n.listeners = append(n.listeners, listenerChannel)
	last := n.lastNotification
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// ignore?
			}
		}()
		listenerChannel <- last
	}()
	return listenerChannel
}

func (n *ApparatchikNotifier) RemoveListener(listenerChannel chan []string) {
	n.Lock()
	defer n.Unlock()
	filtered := []chan []string{}
	for _, existing := range n.listeners {
		if existing != listenerChannel {
			filtered = append(filtered, existing)
		}
	}
	n.listeners = filtered
	close(listenerChannel)
}

func (n *ApparatchikNotifier) Close() {
	n.Lock()
	defer n.Unlock()
	for _, listener := range n.listeners {
		close(listener)
	}
}

func (n *ApparatchikNotifier) NumberOfListeners() int {
	n.Lock()
	defer n.Unlock()
	return len(n.listeners)
}