package profile

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gibson042/canonicaljson-go"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"io"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
	"time"
)

type Service struct {
	relationshipRepo storage.RelationshipRepository
	ctx              context.Context
	appState         *core.AppState
	bus              *bus.EventBus
}

// NewProtocolHandler creates a new chat protocol handler
func NewProtocolHandler(
	app *core.AppState,
	bus *bus.EventBus,
	ctx context.Context,
	repo storage.RelationshipRepository,
) *Service {
	return &Service{
		appState:         app,
		bus:              bus,
		ctx:              ctx,
		relationshipRepo: repo,
	}
}

func (s *Service) Register() {
	log.Printf("Registering Friend Request protocol handler (%s)...", core.FriendRequestProtocolID)
	(*s.appState.Node).SetStreamHandler(core.FriendRequestProtocolID, s.handleFriendRequestStream)
	(*s.appState.Node).SetStreamHandler(core.FriendResponseProtocolID, s.handleFriendResponseStream)
}

func (s *Service) handleFriendRequestStream(stream network.Stream) {
	remotePeerId := stream.Conn().RemotePeer()

	log.Printf("Friend Request: Received friend request from %s", remotePeerId.String())

	receivedBytes, err := io.ReadAll(stream)

	if err != nil {
		log.Printf("Friend Request Handler: Error reading message content from %s: %v", remotePeerId.String(), err)
		stream.Reset()
		return
	}

	var request types.FriendRequest
	err = json.Unmarshal(receivedBytes, &request)

	if err != nil {
		log.Printf("Friend Request Handler: Error deserializing message content from %s: %v", remotePeerId.String(), err)
		stream.Reset()
		return
	}

	log.Printf("Friend Request Handler: Received friend request: %s", receivedBytes)

	requesterPubKey := (*s.appState.Node).Peerstore().PubKey(remotePeerId)

	bytesToVerify, err := canonicaljson.Marshal(request.Data)

	if err != nil {
		log.Printf("Friend Request Handler: ERROR - Failed to create canonical bytes for verification: %v", err)
		return
	}

	isValid, err := requesterPubKey.Verify(bytesToVerify, request.SenderSignature)

	if err != nil {
		log.Printf("Friend Request Handler: ERROR - Signature verification technical error from %s: %v", remotePeerId, err)
		return
	}

	if !isValid {
		log.Printf("Friend Request Handler: https://youtu.be/ckmGncX-MXU?si=gIAUElpsjdiPnPet")
		return
	}

	log.Printf("Friend Request Handler: Signature verified successfully from %s", remotePeerId.String())

	s.bus.PublishAsync(events.FriendRequestReceived{FriendRequest: request.Data})
}

func (s *Service) handleFriendResponseStream(stream network.Stream) {
	remotePeerId := stream.Conn().RemotePeer()

	log.Printf("Friend Response: Received friend response from %s", remotePeerId.String())

	receivedBytes, err := io.ReadAll(stream)

	if err != nil {
		log.Printf("Friend Response Handler: Error reading message content from %s: %v", remotePeerId.String(), err)
		stream.Reset()
		return
	}

	var request types.FriendResponse
	err = json.Unmarshal(receivedBytes, &request)

	if err != nil {
		log.Printf("Friend Response Handler: Error deserializing message content from %s: %v", remotePeerId.String(), err)
		stream.Reset()
		return
	}

	log.Printf("Friend Response Handler: Received friend response: %s", receivedBytes)

	requesterPubKey := (*s.appState.Node).Peerstore().PubKey(remotePeerId)

	bytesToVerify, err := canonicaljson.Marshal(request.Data)

	if err != nil {
		log.Printf("Friend Response Handler: ERROR - Failed to create canonical bytes for verification: %v", err)
		return
	}

	isValid, err := requesterPubKey.Verify(bytesToVerify, request.SenderSignature)

	if err != nil {
		log.Printf("Friend Response Handler: ERROR - Signature verification technical error from %s: %v", remotePeerId, err)
		return
	}

	if !isValid {
		log.Printf("Friend Response Handler: https://youtu.be/ckmGncX-MXU?si=gIAUElpsjdiPnPet")
		return
	}

	log.Printf("Friend Response Handler: Signature verified successfully from %s", remotePeerId.String())

	s.bus.PublishAsync(events.FriendResponseReceivedEvent{
		SenderPeerId: request.Data.ResponderPeerID,
		IsAccepted:   request.Data.IsApproved,
		Timestamp:    request.Data.Timestamp,
	})
}

func (s *Service) SendFriendRequest(receiverPeerId string) error {
	targetPID, err := peer.Decode(receiverPeerId)
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid target PeerID format: %v", err))
	}

	log.Printf("%v", s.appState.State)
	log.Printf("%v", s.appState.Node)
	if s.appState.State != core.StateRunning || s.appState.Node == nil {
		return errors.New(fmt.Sprintf(fmt.Sprintf("Node is not ready (state: %s)", s.appState.State)))
	}

	if targetPID == (*s.appState.Node).ID() {
		return errors.New(fmt.Sprintf("Cannot send friend request to self"))
	}

	data := types.FriendRequestData{
		SenderPeerID: (*s.appState.Node).ID().String(),
		Timestamp:    time.Now().String(),
	}

	bytesToSign, err := canonicaljson.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal canonical json for signing: %w", err)
	}

	senderSignature, err := s.appState.PrivKey.Sign(bytesToSign)

	request := types.FriendRequest{
		Data:            data,
		SenderSignature: senderSignature,
	}

	requestBytes, err := json.Marshal(request)

	log.Printf("Friend request API: Checking connectedness to %s", targetPID.ShortString())
	connectedness := (*s.appState.Node).Network().Connectedness(targetPID)

	if connectedness != network.Connected {
		log.Printf("Friend request API: Not connected to %s (State: %s). Attempting connection...", targetPID.ShortString(), connectedness)

		addrInfo := (*s.appState.Node).Peerstore().PeerInfo(targetPID)
		if len(addrInfo.Addrs) == 0 {
			log.Printf("Friend request API: No addresses found in Peerstore for %s. Cannot connect.", targetPID.ShortString())
			return errors.New(fmt.Sprintf("Cannot connect to peer %s: No known addresses", targetPID.ShortString()))
		}

		// Use a separate context and timeout for the connection attempt
		connectCtx, connectCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer connectCancel()

		err = (*s.appState.Node).Connect(connectCtx, addrInfo)
		if err != nil {
			log.Printf("Friend request API: Failed to connect to %s: %v", targetPID.ShortString(), err)
			return errors.New(fmt.Sprintf("Failed to establish connection with peer %s: %v", targetPID.ShortString(), err))
		}
		log.Printf("Friend request API: Successfully connected to %s.", targetPID.ShortString())
	} else {
		log.Printf("Friend request API: Already connected to %s.", targetPID.ShortString())
	}

	log.Printf("Friend request API: Attempting to open stream to %s for protocol %s", targetPID.ShortString(), core.FriendRequestProtocolID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ctx = network.WithAllowLimitedConn(ctx, "mito")
	defer cancel()

	stream, err := (*s.appState.Node).NewStream(ctx, targetPID, core.FriendRequestProtocolID)
	if err != nil {
		log.Printf("Friend request API: Failed to open stream to %s: %v", targetPID.ShortString(), err)
		return errors.New(fmt.Sprintf("Failed to connect/open stream to peer %s: %v", targetPID.ShortString(), err))
	}
	log.Printf("Friend request API: Stream opened successfully to %s", targetPID.ShortString())

	writer := bufio.NewWriter(stream)
	_, err = writer.Write(requestBytes)

	if err == nil {
		err = writer.Flush()
		stream.CloseWrite()
	}

	if err != nil {
		log.Printf("Error writing/closing friend request stream to %s: %v", receiverPeerId, err)
		stream.Reset() // Reset on error
		return fmt.Errorf("failed to send/close friend request: %w", err)
	}

	s.bus.PublishAsync(events.FriendRequestSentEvent{ReceiverPeerId: receiverPeerId, Timestamp: time.Now()})
	log.Printf("Friend request sent successfully to %s", receiverPeerId)
	return nil
}

func (s *Service) SendFriendResponse(receiverPeerId string, isApproved bool) error {
	targetPID, err := peer.Decode(receiverPeerId)
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid target PeerID format: %v", err))
	}

	data := types.FriendResponseData{
		ResponderPeerID: (*s.appState.Node).ID().String(),
		IsApproved:      isApproved,
		Timestamp:       time.Now().String(),
	}

	bytesToSign, err := canonicaljson.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal canonical json for signing: %w", err)
	}

	senderSignature, err := s.appState.PrivKey.Sign(bytesToSign)

	request := types.FriendResponse{
		Data:            data,
		SenderSignature: senderSignature,
	}

	requestBytes, err := json.Marshal(request)

	log.Printf("Friend response API: Checking connectedness to %s", targetPID.ShortString())
	connectedness := (*s.appState.Node).Network().Connectedness(targetPID)

	if connectedness != network.Connected {
		log.Printf("Friend response API: Not connected to %s (State: %s). Attempting connection...", targetPID.ShortString(), connectedness)

		addrInfo := (*s.appState.Node).Peerstore().PeerInfo(targetPID)
		if len(addrInfo.Addrs) == 0 {
			log.Printf("Friend response API: No addresses found in Peerstore for %s. Cannot connect.", targetPID.ShortString())
			return errors.New(fmt.Sprintf("Cannot connect to peer %s: No known addresses", targetPID.ShortString()))
		}

		// Use a separate context and timeout for the connection attempt
		connectCtx, connectCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer connectCancel()

		err = (*s.appState.Node).Connect(connectCtx, addrInfo)
		if err != nil {
			log.Printf("Friend response API: Failed to connect to %s: %v", targetPID.ShortString(), err)
			return errors.New(fmt.Sprintf("Failed to establish connection with peer %s: %v", targetPID.ShortString(), err))
		}
		log.Printf("Friend response API: Successfully connected to %s.", targetPID.ShortString())
	} else {
		log.Printf("Friend response API: Already connected to %s.", targetPID.ShortString())
	}

	log.Printf("Friend response API: Attempting to open stream to %s for protocol %s", targetPID.ShortString(), core.FriendRequestProtocolID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ctx = network.WithAllowLimitedConn(ctx, "mito")
	defer cancel()

	stream, err := (*s.appState.Node).NewStream(ctx, targetPID, core.FriendResponseProtocolID)
	if err != nil {
		log.Printf("Friend response API: Failed to open stream to %s: %v", targetPID.ShortString(), err)
		return errors.New(fmt.Sprintf("Failed to connect/open stream to peer %s: %v", targetPID.ShortString(), err))
	}
	log.Printf("Friend response API: Stream opened successfully to %s", targetPID.ShortString())

	writer := bufio.NewWriter(stream)
	_, err = writer.Write(requestBytes)

	if err == nil {
		err = writer.Flush()
		stream.CloseWrite()
	}

	if err != nil {
		log.Printf("Error writing/closing friend response stream to %s: %v", receiverPeerId, err)
		stream.Reset() // Reset on error
		return fmt.Errorf("failed to send/close friend response: %w", err)
	}

	log.Printf("Friend response sent successfully to %s", receiverPeerId)
	return nil
}

func (s *Service) RespondToFriendRequest(receiverPeerId string, isAccepted bool) error {
	storeCtx, cancel := context.WithTimeout(s.ctx, 5*time.Second) // Short timeout for DB operation
	defer cancel()

	var status types.FriendStatus
	if isAccepted {
		status = types.FriendStatusApproved
	} else {
		status = types.FriendStatusRejected
	}

	err := s.relationshipRepo.UpdateStatus(storeCtx, types.FriendRelationship{
		PeerID:     receiverPeerId,
		Status:     status,
		ApprovedAt: time.Now(),
	})

	if err != nil {
		return fmt.Errorf("failed to update friend relationship status: %w", err)
	}

	s.bus.PublishAsync(events.FriendResponseSentEvent{PeerId: receiverPeerId, IsAccepted: isAccepted})

	return nil
}

func (s *Service) IsFriend(peerId string) (bool, error) {
	r, err := s.relationshipRepo.GetRelationByPeerId(s.ctx, peerId)

	if err != nil {
		return false, err
	}

	if r.Status == types.FriendStatusApproved {
		return true, nil
	}

	return false, nil
}

func (s *Service) GetFriends() ([]types.FriendRelationship, error) {
	r, err := s.relationshipRepo.GetAcceptedRelations(s.ctx)

	if err != nil {
		log.Printf("Error getting friends: %v", err)
		return nil, err
	}

	return r, nil
}
