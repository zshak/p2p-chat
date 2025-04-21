package api

import (
	"context"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
)

type Consumer struct {
	apiHandler *ApiHandler
	ctx        context.Context
	bus        *bus.EventBus
	eventsChan chan interface{}
}

func NewConsumer(eventBus *bus.EventBus, handler *ApiHandler, ctx context.Context) *Consumer {
	return &Consumer{
		bus:        eventBus,
		eventsChan: make(chan interface{}),
		ctx:        ctx,
		apiHandler: handler,
	}
}

func (c *Consumer) Start() {
	log.Println("api consumer started")
	c.bus.Subscribe(c.eventsChan, events.MessageReceivedEvent{})

	go c.listen()
}

func (c *Consumer) listen() {
	for {
		select {
		case <-c.ctx.Done():
			log.Println("app state observer job stopped")
			return

		case event := <-c.eventsChan:
			c.handleEvent(event)
		}
	}
}

func (c *Consumer) handleEvent(event interface{}) {
	switch ev := event.(type) {

	case events.MessageReceivedEvent:
		log.Println("CONSUMER: received message received event")
		c.apiHandler.send(ev.Message.Content)
		return
	}
}
