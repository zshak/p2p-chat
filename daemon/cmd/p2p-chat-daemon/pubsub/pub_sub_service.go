package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/identity"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
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
	ctx                  context.Context
	appState             *core.AppState
	pubsub               *pubsub.PubSub
	topics               map[string]*pubsub.Topic
	subs                 map[string]*pubsub.Subscription
	groupKeyStoreService *identity.GroupKeyStore
	eventBus             *bus.EventBus
}

// NewPubSubService creates a new pubsub service
func NewPubSubService(
	bus *bus.EventBus,
	ctx context.Context,
	appState *core.AppState,
	groupKeyStoreService *identity.GroupKeyStore,
) (*Service, error) {
	return &Service{
		ctx:                  ctx,
		appState:             appState,
		topics:               make(map[string]*pubsub.Topic),
		subs:                 make(map[string]*pubsub.Subscription),
		groupKeyStoreService: groupKeyStoreService,
		eventBus:             bus,
	}, nil
}

// Start initializes the pubsub topics and subscriptions
func (s *Service) Start() error {
	// Create a new PubSub service using GossipSub
	pubsubService, err := pubsub.NewGossipSub(s.ctx, *s.appState.Node)
	if err != nil {
		return fmt.Errorf("failed to create pubsub service: %w", err)
	}

	s.pubsub = pubsubService

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
	go s.handleIncomingMessages(sub, "")

	return nil
}

// JoinTopic joins pubsub topic
func (s *Service) JoinTopic(topicName string, groupId string) error {
	// Join the online announcement topicName
	topic, err := s.pubsub.Join(topicName)
	if err != nil {
		return fmt.Errorf("failed to join online announcement topicName: %w", err)
	}

	s.topics[topicName] = topic

	// Subscribe to the online announcement topicName
	sub, err := topic.Subscribe()
	if err != nil {
		log.Printf("Error subscribing to online announcement topicName: %v", err)
		return fmt.Errorf("failed to subscribe to online announcement topicName: %w", err)
	}

	// Start listening for messages
	go s.handleIncomingMessages(sub, groupId)

	return nil
}

// handleIncomingMessages processes incoming online announcements
func (s *Service) handleIncomingMessages(sub *pubsub.Subscription, groupId string) {
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

		bytes, _ := s.groupKeyStoreService.Decrypt(groupId, msg.Data)

		var message types.GroupChatMessage
		if err := json.Unmarshal(bytes, &message); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		sender := msg.GetFrom()

		if sender.String() != message.SenderPeerId {
			log.Printf("Matyvilebs vigac: %s != %s", message.SenderPeerId, sender.String())
			continue
		}

		log.Printf("📢📢📢📢📢 MOVIDA: message - %s, dro - %s) 📢📢📢📢📢", message.Message, message.Time)

		mes := events.GroupChatMessage{
			GroupId:      groupId,
			Message:      message.Message,
			SenderPeerId: sender.String(),
			Time:         time.Now(),
		}
		s.eventBus.PublishAsync(events.GroupChatMessageReceivedEvent{Message: mes})
	}
}

// Publish publishes to topic
func (s *Service) Publish(message []byte, topicName string) error {
	for {
		dht := s.appState.Dht
		peerCount := len(dht.RoutingTable().ListPeers())

		_, ok := s.topics[topicName]
		if !ok {
			log.Printf("online announcement topicName not joined")
			continue
		}

		if peerCount > 1 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	topic, ok := s.topics[topicName]
	if !ok {
		return fmt.Errorf("topicName not joined")
	}

	err := topic.Publish(s.ctx, message)

	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	log.Printf("Published message to topic: %s", topicName)
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
