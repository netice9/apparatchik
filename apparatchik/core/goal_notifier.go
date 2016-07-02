/*
* CODE GENERATED AUTOMATICALLY WITH github.com/ernesto-jimenez/gogen/specific
* THIS FILE SHOULD NOT BE EDITED BY HAND
 */

package core

import "sync"

type GoalNotifier struct {
	sync.Mutex
	listeners        []chan GoalStatus
	lastNotification GoalStatus
}

func NewGoalNotifier(firstNotification GoalStatus) *GoalNotifier {
	return &GoalNotifier{
		lastNotification: firstNotification,
	}
}

func (n *GoalNotifier) Notify(value GoalStatus) {
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

func (n *GoalNotifier) AddListener(capacity int) chan GoalStatus {
	listenerChannel := make(chan GoalStatus, capacity)
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

func (n *GoalNotifier) RemoveListener(listenerChannel chan GoalStatus) {
	n.Lock()
	defer n.Unlock()
	filtered := []chan GoalStatus{}
	for _, existing := range n.listeners {
		if existing != listenerChannel {
			filtered = append(filtered, existing)
		}
	}
	n.listeners = filtered
	close(listenerChannel)
}

func (n *GoalNotifier) Close() {
	n.Lock()
	defer n.Unlock()
	for _, listener := range n.listeners {
		close(listener)
	}
}

func (n *GoalNotifier) NumberOfListeners() int {
	n.Lock()
	defer n.Unlock()
	return len(n.listeners)
}
