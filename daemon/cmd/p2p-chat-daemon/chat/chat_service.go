package chat

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gibson042/canonicaljson-go"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"io"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/identity"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/crypto_utils"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/profile"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/pubsub"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
	"strings"
	"sync"
	"time"
)

// Service manages the chat protocol
type Service struct {
	appState             *core.AppState
	bus                  *bus.EventBus
	profileService       *profile.Service
	groupKeyStoreService *identity.GroupKeyStore
	groupMemberRepo      storage.GroupMemberRepository
	KeyRepository        storage.KeyRepository
	messageRepository    storage.MessageRepository
	pubSubService        *pubsub.Service
	groupChats           map[string][]string
	mu                   sync.Mutex
}

// NewProtocolHandler creates a new chat protocol handler
func NewProtocolHandler(
	app *core.AppState,
	bus *bus.EventBus,
	profile *profile.Service,
	groupKeyStore *identity.GroupKeyStore,
	groupMemberRepo storage.GroupMemberRepository,
	keyRepo storage.KeyRepository,
	pubSubService *pubsub.Service,
	messageRepo storage.MessageRepository) *Service {

	return &Service{
		appState:             app,
		bus:                  bus,
		profileService:       profile,
		groupKeyStoreService: groupKeyStore,
		groupMemberRepo:      groupMemberRepo,
		KeyRepository:        keyRepo,
		pubSubService:        pubSubService,
		messageRepository:    messageRepo,
	}
}

// Register registers the chat protocol handler with the node
func (s *Service) Register() {
	log.Printf("Registering chat protocol handler (%s)...", core.ChatProtocolID)

	(*s.appState.Node).SetStreamHandler(core.ChatProtocolID, s.handleChatStream)
	(*s.appState.Node).SetStreamHandler(core.GroupChatProtocolID, s.handleGroupRequest)

	s.startListeningToGroupChatMessages()
}

func (s *Service) startListeningToGroupChatMessages() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	groupChats, err := s.groupMemberRepo.GetGroupsWithMembers(ctx)
	if err != nil {
		log.Printf("Error getting group chats: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.groupChats = groupChats

	for id, _ := range s.groupChats {
		log.Printf("Starting to listen to group chat messages for topic %s", core.GroupChatTopic+id)
		s.pubSubService.JoinTopic(core.GroupChatTopic+id, id)
	}
}

// handleChatStream processes incoming chat streams
func (s *Service) handleChatStream(stream network.Stream) {
	peerID := stream.Conn().RemotePeer()
	log.Printf("Chat: Received new stream from %s", peerID.ShortString())

	isFriend, _ := s.profileService.IsFriend(peerID.String())

	if !isFriend {
		log.Printf("Chat: Received new stream from %s, but they are not a friends. Closing...", peerID.ShortString())
		return
	}

	reader := bufio.NewReader(stream)

	var messageLen uint32
	err := binary.Read(reader, binary.BigEndian, &messageLen)
	if err != nil {
		log.Printf("Chat Handler: Error reading length prefix from %s: %v", peerID.ShortString(), err)
		stream.Reset()
		return
	}

	messageBytes := make([]byte, messageLen)
	_, err = io.ReadFull(reader, messageBytes)
	if err != nil {
		log.Printf("Chat Handler: Error reading message content (expected %d bytes) from %s: %v", messageLen, peerID.ShortString(), err)
		stream.Reset()
		return
	}

	message := string(messageBytes)

	// Trim trailing newline
	message = strings.TrimSpace(message)

	log.Printf("Chat: Received message from %s: <<< %s >>>", peerID.ShortString(), message)

	messageEvent := types.ChatMessage{
		RecipientPeerId: (*s.appState.Node).ID().String(),
		SenderPeerID:    peerID.String(),
		Content:         message,
		SendTime:        time.Now(),
		IsOutgoing:      false,
	}
	s.bus.PublishAsync(events.MessageReceivedEvent{Message: messageEvent})

	// Alternatively, the sender could close it.
	stream.Close()
}

// SendMessage sends a chat message to a peer
func (s *Service) SendMessage(targetPeerId string, message string) error {
	targetPID, err := peer.Decode(targetPeerId)
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid target PeerID format: %v", err))
	}

	if s.appState.State != core.StateRunning || s.appState.Node == nil {
		return errors.New(fmt.Sprintf(fmt.Sprintf("Node is not ready (state: %s)", s.appState.State)))
	}

	if targetPID == (*s.appState.Node).ID() {
		return errors.New(fmt.Sprintf("Cannot send chat message to self"))
	}

	log.Printf("Chat API: Checking connectedness to %s", targetPID.ShortString())
	connectedness := (*s.appState.Node).Network().Connectedness(targetPID)

	if connectedness != network.Connected {
		log.Printf("Chat API: Not connected to %s (State: %s). Attempting connection...", targetPID.ShortString(), connectedness)

		addrInfo := (*s.appState.Node).Peerstore().PeerInfo(targetPID)
		if len(addrInfo.Addrs) == 0 {
			log.Printf("Chat API: No addresses found in Peerstore for %s. Cannot connect.", targetPID.ShortString())
			return errors.New(fmt.Sprintf("Cannot connect to peer %s: No known addresses", targetPID.ShortString()))
		}

		// Use a separate context and timeout for the connection attempt
		connectCtx, connectCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer connectCancel()

		err = (*s.appState.Node).Connect(connectCtx, addrInfo)
		if err != nil {
			log.Printf("Chat API: Failed to connect to %s: %v", targetPID.ShortString(), err)
			return errors.New(fmt.Sprintf("Failed to establish connection with peer %s: %v", targetPID.ShortString(), err))
		}
		log.Printf("Chat API: Successfully connected to %s.", targetPID.ShortString())
	} else {
		log.Printf("Chat API: Already connected to %s.", targetPID.ShortString())
	}

	log.Printf("Chat API: Attempting to open stream to %s for protocol %s", targetPID.ShortString(), core.ChatProtocolID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ctx = network.WithAllowLimitedConn(ctx, "mito")
	defer cancel()

	stream, err := (*s.appState.Node).NewStream(ctx, targetPID, core.ChatProtocolID)
	if err != nil {
		log.Printf("Chat API: Failed to open stream to %s: %v", targetPID.ShortString(), err)
		return errors.New(fmt.Sprintf("Failed to connect/open stream to peer %s: %v", targetPID.ShortString(), err))
	}
	log.Printf("Chat API: Stream opened successfully to %s", targetPID.ShortString())

	// --- Send Message ---
	messageBytes := []byte(message)
	messageLen := uint32(len(messageBytes))

	writer := bufio.NewWriter(stream)

	err = binary.Write(writer, binary.BigEndian, messageLen)
	if err != nil {
		stream.Reset()
		return fmt.Errorf("failed to write message length prefix: %w", err)
	}

	_, err = writer.Write(messageBytes)
	if err != nil {
		stream.Reset()
		return fmt.Errorf("failed to write message content: %w", err)
	}

	err = writer.Flush()
	if err != nil {
		stream.Reset()
		return fmt.Errorf("failed to flush stream writer: %w", err)
	}

	messageEvent := types.ChatMessage{
		RecipientPeerId: targetPeerId,
		SenderPeerID:    (*s.appState.Node).ID().String(),
		Content:         message,
		SendTime:        time.Now(),
		IsOutgoing:      true,
	}

	s.bus.PublishAsync(events.MessageSentEvent{Message: messageEvent})
	log.Printf("Chat API: Message sent successfully to %s", targetPID.ShortString())

	// --- Close Stream (an ara ar vici gadasawyvetia) ---
	// Closing the stream signals the other side we're done writing.
	// Our simple receiver closes after reading one line anyway.
	err = stream.Close()
	if err != nil {
		log.Printf("Chat API: Error closing stream to %s: %v", targetPID.ShortString(), err)
	}

	return nil
}

// CreateGroup creates a group
func (s *Service) CreateGroup(peers []string, groupChatName string) error {
	id := uuid.New().String()

	k, e := s.groupKeyStoreService.GenerateNewKey(id)

	if e != nil {
		log.Printf("GROUP Chat API: Error generating new key: %v", e)
		return e
	}

	req := GroupChatRequest{
		MemberPeers: peers,
		Name:        groupChatName,
		Key:         k,
		Id:          id,
	}

	requestBytes, err := canonicaljson.Marshal(req)

	if err != nil {
		log.Printf("GROUP Chat API: Error serializing request: %v", err)
		return err
	}

	for _, p := range peers {
		targetPID, err := peer.Decode(p)
		if err != nil {
			return errors.New(fmt.Sprintf("Invalid target PeerID format: %v", err))
		}

		connectedness := (*s.appState.Node).Network().Connectedness(targetPID)

		if connectedness != network.Connected {
			log.Printf("GROUP Chat API: Not connected to %s (State: %s). Attempting connection...", targetPID.ShortString(), connectedness)

			addrInfo := (*s.appState.Node).Peerstore().PeerInfo(targetPID)
			if len(addrInfo.Addrs) == 0 {
				log.Printf("GROUP Chat API: No addresses found in Peerstore for %s. Cannot connect.", targetPID.ShortString())
				return errors.New(fmt.Sprintf("Cannot connect to peer %s: No known addresses", targetPID.ShortString()))
			}

			// Use a separate context and timeout for the connection attempt
			connectCtx, connectCancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer connectCancel()

			err = (*s.appState.Node).Connect(connectCtx, addrInfo)
			if err != nil {
				log.Printf("GROUP Chat API: Failed to connect to %s: %v", targetPID.ShortString(), err)
				return errors.New(fmt.Sprintf("Failed to establish connection with peer %s: %v", targetPID.ShortString(), err))
			}
			log.Printf("GROUP Chat API: Successfully connected to %s.", targetPID.ShortString())
		} else {
			log.Printf("GROUP Chat API: Already connected to %s.", targetPID.ShortString())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		ctx = network.WithAllowLimitedConn(ctx, "mito")
		defer cancel()

		stream, err := (*s.appState.Node).NewStream(ctx, targetPID, core.GroupChatProtocolID)
		if err != nil {
			log.Printf("GROUP Chat API: Failed to open stream to %s: %v", targetPID.ShortString(), err)
			return errors.New(fmt.Sprintf("Failed to connect/open stream to peer %s: %v", targetPID.ShortString(), err))
		}
		log.Printf("GROUP Chat API: Stream opened successfully to %s", targetPID.ShortString())

		// --- Send Group Chat Request ---
		writer := bufio.NewWriter(stream)

		_, err = writer.Write(requestBytes)
		if err != nil {
			stream.Reset()
			return fmt.Errorf("failed to write message content: %w", err)
		}
		err = writer.Flush()

		if err != nil {
			stream.Reset()
			return fmt.Errorf("failed to flush stream writer: %w", err)
		}

		stream.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = s.groupMemberRepo.AddMembers(ctx, id, peers)

	if err != nil {
		log.Printf("GROUP Chat API: Error adding members to group: %v", err)
		return err
	}

	log.Printf("GROUP Chat API: joining topic: %s", core.GroupChatTopic+id)
	err = s.pubSubService.JoinTopic(core.GroupChatTopic+id, id)

	if err != nil {
		log.Printf("GROUP Chat API: Error joining topic: %v", err)
		return err
	}

	return nil
}

// handleGroupRequest processes incoming group creation request
func (s *Service) handleGroupRequest(stream network.Stream) {
	peerID := stream.Conn().RemotePeer()
	log.Printf("GroupRequest: Received new stream from %s", peerID.ShortString())

	receivedBytes, err := io.ReadAll(stream)

	if err != nil {
		log.Printf("Group Request Handler: Error reading group request from %s: %v", peerID.String(), err)
		stream.Reset()
		return
	}

	var request GroupChatRequest
	err = json.Unmarshal(receivedBytes, &request)

	if err != nil {
		log.Printf("Group Request Handler: Error deserializing group request from %s: %v", peerID.String(), err)
		stream.Reset()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = s.KeyRepository.Store(ctx, types.GroupKey{
		GroupId:   request.Id,
		Key:       request.Key,
		CreatedAt: time.Now(),
	})

	if err != nil {
		log.Printf("Group Request Handler: Error storing group key: %v", err)
	}

	err = s.groupMemberRepo.AddMembers(ctx, request.Id, request.MemberPeers)

	if err != nil {
		log.Printf("Group Request Handler: Error adding members to group: %v", err)
	}

	log.Printf("GROUP Chat API: joining topic: %s", core.GroupChatTopic+request.Id)

	err = s.pubSubService.JoinTopic(core.GroupChatTopic+request.Id, request.Id)
	if err != nil {
		log.Printf("GROUP Chat API: Error joining topic: %v", err)
	}

	stream.Close()
}

func (s *Service) SendGroupMessage(groupId string, message string) error {
	messageID, _ := uuid.NewRandom()

	pubSubMessage := types.GroupChatMessage{
		SenderPeerId: (*s.appState.Node).ID().String(),
		Message:      message,
		Time:         time.Now(),
		Id:           messageID.String(),
	}

	pubSubMessageBytes, err := json.Marshal(pubSubMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	encryptedMessage, err := s.groupKeyStoreService.Encrypt(groupId, pubSubMessageBytes)

	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	s.pubSubService.Publish(encryptedMessage, core.GroupChatTopic+groupId)

	mes := events.GroupChatMessage{
		GroupId:      groupId,
		Message:      message,
		SenderPeerId: (*s.appState.Node).ID().String(),
		Time:         time.Now(),
	}
	s.bus.PublishAsync(events.GroupChatMessageSentEvent{Message: mes})

	return nil
}

func (s *Service) GetGroupMessages(groupId string) (GroupChatMessages, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messages, err := s.messageRepository.GetGroupMessages(ctx, groupId, 1000000, time.Now())

	if err != nil {
		return GroupChatMessages{}, err
	}

	groupChatMessages := make([]GroupChatMessage, 0)

	for _, m := range messages {
		decryptedMessage, err := crypto_utils.DecryptDataWithKey(
			s.appState.DbKey,
			m.EncryptedContent,
			core.DefaultCryptoConfig,
		)

		if err != nil {
			log.Printf("Error decrypting message: %v", err)
			continue
		}

		groupChatMessages = append(groupChatMessages, GroupChatMessage{
			SenderPeerId: m.SenderPeerID,
			Time:         m.SentAt,
			Message:      string(decryptedMessage),
		})
	}
	return GroupChatMessages{Messages: groupChatMessages}, nil
}

func (s *Service) GetMessages(peerId string) (Messages, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messages, err := s.messageRepository.GetMessagesByPeerID(ctx, peerId, 1000000)

	if err != nil {
		return Messages{}, err
	}

	groupChatMessages := make([]Message, 0)

	for _, m := range messages {
		decryptedMessage, err := crypto_utils.DecryptDataWithKey(
			s.appState.DbKey,
			m.Content,
			core.DefaultCryptoConfig,
		)

		if err != nil {
			log.Printf("Error decrypting message: %v", err)
			continue
		}

		groupChatMessages = append(groupChatMessages, Message{
			SendTime:   m.SendTime,
			Message:    string(decryptedMessage),
			IsOutgoing: m.IsOutgoing,
		})
	}
	return Messages{Messages: groupChatMessages}, nil
}

func (s *Service) GetGroups() ([]storage.GroupInfo, error) {
	groups, err := s.groupMemberRepo.GetGroups(context.Background())

	if err != nil {
		return nil, err
	}

	return groups, nil
}
