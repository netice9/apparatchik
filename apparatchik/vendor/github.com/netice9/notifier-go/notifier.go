package notifier

import "sync"

type Notifier struct {
	sync.Mutex
	listeners        []chan interface{}
	lastNotification *interface{}
}

func NewNotifier() *Notifier {
	return &Notifier{}
}

func (n *Notifier) Notify(value interface{}) {
	n.Lock()
	defer n.Unlock()

	n.lastNotification = &value
	for _, listener := range n.listeners {
		l := listener
		go func() {
			l <- value
		}()
	}

}

func (n *Notifier) AddListener(listenerChannel chan interface{}) {
	n.Lock()
	defer n.Unlock()
	for _, existing := range n.listeners {
		if existing == listenerChannel {
			return
		}
	}
	n.listeners = append(n.listeners, listenerChannel)
	if n.lastNotification != nil {
		last := *n.lastNotification
		go func() {
			listenerChannel <- last
		}()
	}
}

func (n *Notifier) RemoveListener(listenerChannel chan interface{}) {
	n.Lock()
	defer n.Unlock()
	filtered := []chan interface{}{}
	for _, existing := range n.listeners {
		if existing != listenerChannel {
			filtered = append(filtered, existing)
		}
	}
	n.listeners = filtered
}

func (n *Notifier) Close() {
	n.Lock()
	defer n.Unlock()
	for _, listener := range n.listeners {
		close(listener)
	}
}

func (n *Notifier) NumberOfListeners() int {
	n.Lock()
	defer n.Unlock()
	return len(n.listeners)
}
