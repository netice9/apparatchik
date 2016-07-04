/*
* CODE GENERATED AUTOMATICALLY WITH github.com/ernesto-jimenez/gogen/specific
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package core

import "sync"

// ApplicationNotifier is a event broadcaster
type ApplicationNotifier struct {
	sync.Mutex
	listeners		[]chan ApplicationStatus
	lastNotification	ApplicationStatus
}

// NewApplicationNotifier creates a new ApplicationNotifier with initial notfication value
func NewApplicationNotifier(firstNotification ApplicationStatus) *ApplicationNotifier {
	return &ApplicationNotifier{
		lastNotification: firstNotification,
	}
}

// Notify notifies current value to all listeners
func (n *ApplicationNotifier) Notify(value ApplicationStatus) {

	nonBlockingSendToChannel := func(chn chan ApplicationStatus, val ApplicationStatus) {
		// recover in the case of sending to closed channel
		defer func() {
			if r := recover(); r != nil {
				// ignore?
			}
		}()

		select {
		case chn <- val:
			// everything is ok
		default:
			// previous value is blocking the channel, remove it!
			select {
			case <-chn:
				// removed value, all clear to send!
				chn <- val
			default:
				// receiver read it, send it now!
				chn <- val
			}
		}

	}

	n.Lock()
	defer n.Unlock()

	n.lastNotification = value
	for _, listener := range n.listeners {
		nonBlockingSendToChannel(listener, value)
	}

}

// AddListener creats a new listener channel
func (n *ApplicationNotifier) AddListener(capacity int) chan ApplicationStatus {
	if capacity == 0 {
		capacity = 1
	}
	listenerChannel := make(chan ApplicationStatus, capacity)
	n.Lock()
	defer n.Unlock()
	n.listeners = append(n.listeners, listenerChannel)
	last := n.lastNotification
	listenerChannel <- last
	return listenerChannel
}

// RemoveListener removes and closes an existing listener channel
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

// Close closes and removes all listeners
func (n *ApplicationNotifier) Close() {
	n.Lock()
	defer n.Unlock()
	for _, listener := range n.listeners {
		close(listener)
	}
	n.listeners = []chan ApplicationStatus{}
}

// NumberOfListeners returns the current count of open listeners
func (n *ApplicationNotifier) NumberOfListeners() int {
	n.Lock()
	defer n.Unlock()
	return len(n.listeners)
}
