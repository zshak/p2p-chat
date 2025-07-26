package chat

import (
	"context"
	"errors"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/crypto_utils"
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
	c.bus.Subscribe(c.eventsChan, events.GroupChatMessageReceivedEvent{})
	c.bus.Subscribe(c.eventsChan, events.GroupChatMessageSentEvent{})

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

	case events.GroupChatMessageReceivedEvent:
		log.Println("received group chat message received event")
		c.handleGroupChatMessageReceivedEvent(event.Message)
		return

	case events.GroupChatMessageSentEvent:
		log.Println("received group chat message received event")
		c.handleGroupChatMessageSentEvent(event.Message)
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
	storeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	encryptedMessage, err := crypto_utils.EncryptDataWithKey(c.appState.DbKey, []byte(message.Content), core.DefaultCryptoConfig)

	if err != nil {
		log.Printf("Chat Consumer: ERROR - Failed to encrypt message: %v", err)
		return
	}

	id, err := c.chatRepo.Store(storeCtx, types.StoredMessage{
		SenderPeerID:    message.SenderPeerID,
		RecipientPeerId: message.RecipientPeerId,
		Content:         encryptedMessage,
		SendTime:        message.SendTime,
		IsOutgoing:      message.IsOutgoing,
	})
	if err != nil {
		log.Printf("Chat Consumer: ERROR - Failed to store sent message (ID tentative %d) to %s: %v", id, message.RecipientPeerId, err)
	} else {
		log.Printf("Chat Consumer: Successfully stored sent message with DB ID %d to %s", id, message.RecipientPeerId)
	}
}

func (c *Consumer) handleGroupChatMessageReceivedEvent(event events.GroupChatMessage) {
	c.SaveGroupChatMessage(event)
}

func (c *Consumer) handleGroupChatMessageSentEvent(message events.GroupChatMessage) {
	c.SaveGroupChatMessage(message)
}

func (c *Consumer) SaveGroupChatMessage(event events.GroupChatMessage) {
	storeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	encryptedMesasge, err := crypto_utils.EncryptDataWithKey(c.appState.DbKey, []byte(event.Message), core.DefaultCryptoConfig)

	if err != nil {
		log.Printf("Chat Consumer: ERROR - Failed to encrypt group chat message: %v", err)
		return
	}

	msg := types.StoredGroupMessage{
		GroupID:          event.GroupId,
		SenderPeerID:     event.SenderPeerId,
		EncryptedContent: encryptedMesasge,
		SentAt:           time.Now(),
	}
	err = c.chatRepo.StoreGroupMessage(storeCtx, msg)

	if err != nil {
		log.Printf("Chat Consumer: ERROR - Failed to store group chat message: %v", err)
		return
	}
}
