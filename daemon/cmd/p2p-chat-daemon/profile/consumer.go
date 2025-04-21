package profile

import (
	"context"
	"errors"
	"fmt"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
	"strings"
	"time"
)

type Consumer struct {
	appState         *core.AppState
	bus              *bus.EventBus
	ctx              context.Context
	relationshipRepo storage.RelationshipRepository
	eventsChan       chan interface{}
}

func NewConsumer(appState *core.AppState, eventBus *bus.EventBus, repo storage.RelationshipRepository, ctx context.Context) (*Consumer, error) {
	if appState == nil {
		return nil, errors.New("appState is nil")
	}
	return &Consumer{appState: appState, bus: eventBus, ctx: ctx, relationshipRepo: repo, eventsChan: make(chan interface{})}, nil
}

func (c *Consumer) Start() {
	log.Println("chat consumer started")
	c.bus.Subscribe(c.eventsChan, events.FriendRequestReceived{})

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

	case events.FriendRequestReceived:
		log.Println("received message sent event")
		c.handleFriendRequestReceived(event.FriendRequest)
		return
	}
}

func (c *Consumer) handleFriendRequestReceived(request types.FriendRequestData) {
	storeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second) // Short timeout for DB operation
	defer cancel()

	// Parse the timestamp (excluding the "m=+46.107792917" part)
	layout := "2006-01-02 15:04:05.999999 -0700 MST"

	// Remove the monotonic clock portion (m=+46.107792917)
	cleanTimestamp := request.Timestamp
	if idx := strings.Index(request.Timestamp, " m=+"); idx > 0 {
		cleanTimestamp = request.Timestamp[:idx]
	}

	t, err := time.Parse(layout, cleanTimestamp)
	if err != nil {
		fmt.Println("Error parsing timestamp:", err)
		return
	}

	entity := types.FriendRelationship{
		PeerID:      request.SenderPeerID,
		Status:      types.FriendStatusPending,
		RequestedAt: t,
	}

	err = c.relationshipRepo.Store(storeCtx, entity)
	if err != nil {
		log.Printf("Profile Consumer: ERROR - Failed to store friend request of %s: %v", request.SenderPeerID, err)
	} else {
		log.Printf("Profile Consumer: Successfully stored friend request of %s", request.SenderPeerID)
	}
}
