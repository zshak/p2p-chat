package api

import (
	"context"
	"encoding/json"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
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
	c.bus.Subscribe(c.eventsChan, events.GroupChatMessageReceivedEvent{})
	c.bus.Subscribe(c.eventsChan, events.MessageSentEvent{})
	c.bus.Subscribe(c.eventsChan, events.GroupChatMessageSentEvent{})

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
		c.HandleMessageReceived(ev.Message)
		return

	case events.MessageSentEvent:
		c.HandleMessageReceived(ev.Message)
		return

	case events.GroupChatMessageSentEvent:
		c.HandleGroupMessageReceived(ev.Message)
		return

	case events.GroupChatMessageReceivedEvent:
		c.HandleGroupMessageReceived(ev.Message)
		return
	}
}

func (c *Consumer) HandleMessageReceived(message types.ChatMessage) {
	log.Println("CONSUMER: received message received event")
	wsMsg := WsMessage{
		Type: WsMsgTypeDirectMessage,
	}

	payload := WsDirectMessagePayload{
		TargetPeerId: message.RecipientPeerId,
		SenderPeerId: message.SenderPeerID,
		Message:      message.Content,
	}
	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		log.Printf("ERROR: Failed to marshal payload: %v", err)
		return
	}

	wsMsg.Payload = payloadBytes

	wsMsgBytes, err := json.Marshal(wsMsg)

	c.apiHandler.send(wsMsgBytes)
}

func (c *Consumer) HandleGroupMessageReceived(message events.GroupChatMessage) {
	log.Println("CONSUMER: received group message received event")
	wsMsg := WsMessage{
		Type: WsMsgTypeGroupMessage,
	}

	payload := WsGroupMessagePayload{
		SenderPeerId: message.SenderPeerId,
		Message:      message.Message,
		GroupId:      message.GroupId,
	}
	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		log.Printf("ERROR: Failed to marshal payload: %v", err)
		return
	}

	wsMsg.Payload = payloadBytes

	wsMsgBytes, err := json.Marshal(wsMsg)

	c.apiHandler.send(wsMsgBytes)
}
