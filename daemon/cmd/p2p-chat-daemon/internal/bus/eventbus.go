package bus

import (
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"reflect"
	"sync"
)

type EventBus struct {
	observers map[string][]chan interface{}
	mu        sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		observers: make(map[string][]chan interface{}),
	}
}

func (eb *EventBus) Subscribe(ch chan interface{}, event core.Event) {
	eventType := reflect.TypeOf(event).String()

	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.observers[eventType] = append(eb.observers[eventType], ch)
}

func (eb *EventBus) Publish(event core.Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	observerChans := eb.observers[reflect.TypeOf(event).String()]

	for _, observerChan := range observerChans {
		go func(ch chan interface{}) {
			observerChan <- event
		}(observerChan)
	}
}
