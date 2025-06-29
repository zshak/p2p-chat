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
	profileService   *Service
	eventsChan       chan interface{}
}

func NewConsumer(
	appState *core.AppState,
	eventBus *bus.EventBus,
	repo storage.RelationshipRepository,
	profileService *Service,
	ctx context.Context,
) (*Consumer, error) {
	if appState == nil {
		return nil, errors.New("appState is nil")
	}
	return &Consumer{
		appState:         appState,
		bus:              eventBus,
		ctx:              ctx,
		relationshipRepo: repo,
		profileService:   profileService,
		eventsChan:       make(chan interface{}),
	}, nil
}

func (c *Consumer) Start() {
	log.Println("chat consumer started")
	c.bus.Subscribe(c.eventsChan, events.FriendRequestReceived{})
	c.bus.Subscribe(c.eventsChan, events.FriendRequestSentEvent{})
	c.bus.Subscribe(c.eventsChan, events.FriendResponseSentEvent{})
	c.bus.Subscribe(c.eventsChan, events.FriendResponseReceivedEvent{})

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
		log.Println("received friends request received event")
		c.handleFriendRequestReceived(event.FriendRequest)
		return

	case events.FriendRequestSentEvent:
		log.Println("received friends request sent event")
		c.handleFriendRequestSent(event)
		return

	case events.FriendResponseSentEvent:
		log.Println("received friends response sent event")
		c.handleFriendResponseSentEvent(event)
		return

	case events.FriendResponseReceivedEvent:
		log.Printf("received friends response received event - %s", event)
		c.handleFriendResponseReceivedEvent(event)
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

	curRel, err := c.relationshipRepo.GetRelationByPeerId(storeCtx, request.SenderPeerID)

	// already exists
	if curRel.PeerID != "" {
		log.Printf("Profile Consumer: Received duplicate friends request from %s", request.SenderPeerID)
		return
	}

	err = c.relationshipRepo.Store(storeCtx, entity)
	if err != nil {
		log.Printf("Profile Consumer: ERROR - Failed to store friends request from %s: %v", request.SenderPeerID, err)
	} else {
		log.Printf("Profile Consumer: Successfully stored friends request from %s", request.SenderPeerID)
	}
}

func (c *Consumer) handleFriendRequestSent(event events.FriendRequestSentEvent) {
	storeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second) // Short timeout for DB operation
	defer cancel()

	// Parse the timestamp (excluding the "m=+46.107792917" part)
	layout := "2006-01-02 15:04:05.999999 -0700 MST"

	// Remove the monotonic clock portion (m=+46.107792917)
	cleanTimestamp := event.Timestamp.String()
	if idx := strings.Index(event.Timestamp.String(), " m=+"); idx > 0 {
		cleanTimestamp = event.Timestamp.String()[:idx]
	}

	t, err := time.Parse(layout, cleanTimestamp)
	if err != nil {
		fmt.Println("Error parsing timestamp:", err)
		return
	}

	entity := types.FriendRelationship{
		PeerID:      event.ReceiverPeerId,
		Status:      types.FriendStatusPending,
		RequestedAt: t,
	}

	err = c.relationshipRepo.Store(storeCtx, entity)
	if err != nil {
		log.Printf("Profile Consumer: ERROR - Failed to store friends request sent to %s: %v", event.ReceiverPeerId, err)
	} else {
		log.Printf("Profile Consumer: Successfully stored friends request sent to %s", event.ReceiverPeerId)
	}
}

func (c *Consumer) handleFriendResponseSentEvent(event events.FriendResponseSentEvent) {
	err := c.profileService.SendFriendResponse(
		event.PeerId,
		event.IsAccepted,
	)
	if err != nil {
		log.Printf("Profile Consumer: ERROR - Failed to send friends response to %s: %v", event.PeerId, err)
		return
	}
	log.Printf("Profile Consumer: Successfully sent friends response to %s", event.PeerId)
}

func (c *Consumer) handleFriendResponseReceivedEvent(event events.FriendResponseReceivedEvent) {
	storeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second) // Short timeout for DB operation
	defer cancel()

	status := event.Status

	// Parse the timestamp (excluding the "m=+46.107792917" part)
	layout := "2006-01-02 15:04:05.999999 -0700 MST"

	// Remove the monotonic clock portion (m=+46.107792917)
	cleanTimestamp := event.Timestamp
	if idx := strings.Index(event.Timestamp, " m=+"); idx > 0 {
		cleanTimestamp = event.Timestamp[:idx]
	}

	t, err := time.Parse(layout, cleanTimestamp)
	if err != nil {
		fmt.Println("Error parsing timestamp:", err)
		return
	}

	entity := types.FriendRelationship{
		PeerID:     event.SenderPeerId,
		Status:     status,
		ApprovedAt: t,
	}

	err = c.relationshipRepo.UpdateStatus(storeCtx, entity)
	if err != nil {
		log.Printf("Profile Consumer: ERROR - Failed to store friends response from %s: %v", event.SenderPeerId, err)
	} else {
		log.Printf("Profile Consumer: Successfully stored friends response from %s", event.SenderPeerId)
	}
}
