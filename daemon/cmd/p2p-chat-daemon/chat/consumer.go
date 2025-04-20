package chat

import (
	"context"
	"errors"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
	"time"
)

type Consumer struct {
	appState   *core.AppState
	bus        *bus.EventBus
	ctx        context.Context
	chatRepo   storage.MessageRepository
	eventsChan chan interface{}
}

func NewConsumer(appState *core.AppState, eventBus *bus.EventBus, repo storage.MessageRepository, ctx context.Context) (*Consumer, error) {
	if appState == nil {
		return nil, errors.New("appState is nil")
	}
	return &Consumer{appState: appState, bus: eventBus, ctx: ctx, chatRepo: repo, eventsChan: make(chan interface{})}, nil
}

func (c *Consumer) Start() {
	log.Println("chat consumer started")
	c.bus.Subscribe(c.eventsChan, events.MessageSentEvent{})
	c.bus.Subscribe(c.eventsChan, events.MessageReceivedEvent{})

	go c.listen()
}

func (c *Consumer) listen() {
	for {
		select {
		case <-c.ctx.Done():
			log.Println("chat consumer stopped")
			return

		case event := <-c.eventsChan:
			c.handleEvent(event)
		}
	}
}

func (c *Consumer) handleEvent(event interface{}) {
	switch event := event.(type) {

	case events.MessageSentEvent:
		log.Println("received message sent event")
		c.handleMessageSent(event.Message)
		return

	case events.MessageReceivedEvent:
		log.Println("received message received event")
		c.handleMessageReceived(event.Message)
		return
	}
}

func (c *Consumer) handleMessageSent(message types.ChatMessage) {
	c.SaveMessage(message)
}

func (c *Consumer) handleMessageReceived(message types.ChatMessage) {
	c.SaveMessage(message)
}

func (c *Consumer) SaveMessage(message types.ChatMessage) {
	storeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second) // Short timeout for DB operation
	defer cancel()

	id, err := c.chatRepo.Store(storeCtx, message)
	if err != nil {
		log.Printf("Chat Consumer: ERROR - Failed to store sent message (ID tentative %d) to %s: %v", id, message.RecipientPeerId, err)
	} else {
		log.Printf("Chat Consumer: Successfully stored sent message with DB ID %d to %s", id, message.RecipientPeerId)
	}
}
