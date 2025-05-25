package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p-pubsub"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
)

const (
	// OnlineAnnouncementTopic Topic for online announcements
	OnlineAnnouncementTopic = core.OnlineAnnouncementTopic

	// MsgTypeOnlineAnnouncement Message types
	MsgTypeOnlineAnnouncement = "online-announcement"
)

// OnlineAnnouncement represents a message announcing a peer is online
type OnlineAnnouncement struct {
	Type      string    `json:"type"`
	PeerID    string    `json:"peer_id"`
	Timestamp time.Time `json:"timestamp"`
	LastMsgID string    `json:"last_msg_id"`
}

// Service manages the pubsub functionality
type Service struct {
	ctx      context.Context
	appState *core.AppState
	pubsub   *pubsub.PubSub
	topics   map[string]*pubsub.Topic
	subs     map[string]*pubsub.Subscription
}

// NewPubSubService creates a new pubsub service
func NewPubSubService(ctx context.Context, appState *core.AppState) (*Service, error) {
	if appState.Node == nil {
		return nil, fmt.Errorf("node not initialized")
	}

	// Create a new PubSub service using GossipSub
	pubsubService, err := pubsub.NewGossipSub(ctx, *appState.Node)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub service: %w", err)
	}

	return &Service{
		ctx:      ctx,
		appState: appState,
		pubsub:   pubsubService,
		topics:   make(map[string]*pubsub.Topic),
		subs:     make(map[string]*pubsub.Subscription),
	}, nil
}

// Start initializes the pubsub topics and subscriptions
func (s *Service) Start() error {
	// Join the online announcement topic
	onlineTopic, err := s.pubsub.Join(OnlineAnnouncementTopic)
	if err != nil {
		return fmt.Errorf("failed to join online announcement topic: %w", err)
	}
	s.topics[OnlineAnnouncementTopic] = onlineTopic

	// Subscribe to the online announcement topic
	sub, err := onlineTopic.Subscribe()
	if err != nil {
		log.Printf("Error subscribing to online announcement topic: %v", err)
		return fmt.Errorf("failed to subscribe to online announcement topic: %w", err)
	}
	s.subs[OnlineAnnouncementTopic] = sub

	// Start listening for messages
	go s.handleOnlineAnnouncements(sub)

	// Announce that we're online
	go s.announceOnline("")

	return nil
}

// handleOnlineAnnouncements processes incoming online announcements
func (s *Service) handleOnlineAnnouncements(sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(s.ctx)
		if err != nil {
			log.Printf("Error receiving pubsub message: %v", err)
			return
		}

		// Ignore messages from ourselves
		if msg.ReceivedFrom == (*s.appState.Node).ID() {
			continue
		}

		var announcement OnlineAnnouncement
		if err := json.Unmarshal(msg.Data, &announcement); err != nil {
			log.Printf("Error unmarshalling online announcement: %v", err)
			continue
		}

		log.Printf("📢 Peer online announcement: %s (last message ID: %s)",
			announcement.PeerID, announcement.LastMsgID)
	}
}

// AnnounceOnline broadcasts that the current peer is online
func (s *Service) announceOnline(lastMsgID string) error {
	for {
		dht := s.appState.Dht
		peerCount := len(dht.RoutingTable().ListPeers())

		topic, ok := s.topics[OnlineAnnouncementTopic]
		if !ok {
			log.Printf("online announcement topic not joined")
			continue
		}
		peers := topic.ListPeers()

		if peerCount > 1 && len(peers) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	topic, ok := s.topics[OnlineAnnouncementTopic]
	if !ok {
		return fmt.Errorf("online announcement topic not joined")
	}

	announcement := OnlineAnnouncement{
		Type:      MsgTypeOnlineAnnouncement,
		PeerID:    (*s.appState.Node).ID().String(),
		Timestamp: time.Now(),
		LastMsgID: lastMsgID,
	}

	data, err := json.Marshal(announcement)
	if err != nil {
		return fmt.Errorf("failed to marshal online announcement: %w", err)
	}

	err = topic.Publish(s.ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish online announcement: %w", err)
	}

	log.Printf("Published online announcement with last message ID: %s", lastMsgID)
	return nil
}

// Stop cleans up pubsub resources
func (s *Service) Stop() error {
	for name, sub := range s.subs {
		sub.Cancel()
		delete(s.subs, name)
	}

	for name := range s.topics {
		delete(s.topics, name)
	}

	return nil
}
