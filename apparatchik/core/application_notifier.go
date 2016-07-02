/*
* CODE GENERATED AUTOMATICALLY WITH github.com/ernesto-jimenez/gogen/specific
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package core

import "sync"

type ApplicationNotifier struct {
	sync.Mutex
	listeners		[]chan ApplicationStatus
	lastNotification	ApplicationStatus
}

func NewApplicationNotifier(firstNotification ApplicationStatus) *ApplicationNotifier {
	return &ApplicationNotifier{
		lastNotification: firstNotification,
	}
}

func (n *ApplicationNotifier) Notify(value ApplicationStatus) {
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

func (n *ApplicationNotifier) AddListener(capacity int) chan ApplicationStatus {
	listenerChannel := make(chan ApplicationStatus, capacity)
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

func (n *ApplicationNotifier) RemoveListener(listenerChannel chan ApplicationStatus) {
	n.Lock()
	defer n.Unlock()
	filtered := []chan ApplicationStatus{}
	for _, existing := range n.listeners {
		if existing != listenerChannel {
			filtered = append(filtered, existing)
		}
	}
	n.listeners = filtered
	close(listenerChannel)
}

func (n *ApplicationNotifier) Close() {
	n.Lock()
	defer n.Unlock()
	for _, listener := range n.listeners {
		close(listener)
	}
}

func (n *ApplicationNotifier) NumberOfListeners() int {
	n.Lock()
	defer n.Unlock()
	return len(n.listeners)
}
