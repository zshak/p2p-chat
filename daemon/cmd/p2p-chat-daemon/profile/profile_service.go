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
	"p2p-chat-daemon/cmd/p2p-chat-daemon/connection"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
	"time"
)

type Service struct {
	relationshipRepo  storage.RelationshipRepository
	connectionService *connection.Service
	ctx               context.Context
	appState          *core.AppState
	bus               *bus.EventBus
}

// NewProtocolHandler creates a new chat protocol handler
func NewProtocolHandler(
	app *core.AppState,
	bus *bus.EventBus,
	ctx context.Context,
	repo storage.RelationshipRepository,
	connSvc *connection.Service,
) *Service {
	return &Service{
		appState:          app,
		bus:               bus,
		ctx:               ctx,
		relationshipRepo:  repo,
		connectionService: connSvc,
	}
}

func (s *Service) Register() {
	log.Printf("Registering Friend Request protocol handler (%s)...", core.FriendRequestProtocolID)
	(*s.appState.Node).SetStreamHandler(core.FriendRequestProtocolID, s.handleFriendRequestStream)
	(*s.appState.Node).SetStreamHandler(core.FriendResponseProtocolID, s.handleFriendResponseStream)
	(*s.appState.Node).SetStreamHandler(core.FriendResponsePollProtocolId, s.handleFriendResponsePollStream)

	go s.PollForFriendResponse()
}

func (s *Service) handleFriendRequestStream(stream network.Stream) {
	remotePeerId := stream.Conn().RemotePeer()

	log.Printf("Friend Request: Received friends request from %s", remotePeerId.String())

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

	log.Printf("Friend Request Handler: Received friends request: %s", receivedBytes)

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

	log.Printf("Friend Response: Received friends response from %s", remotePeerId.String())

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

	log.Printf("Friend Response Handler: Received friends response: %s", receivedBytes)

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

	var status types.FriendStatus

	if request.Data.IsApproved {
		status = types.FriendStatusApproved
	} else {
		status = types.FriendStatusRejected
	}

	s.bus.PublishAsync(events.FriendResponseReceivedEvent{
		SenderPeerId: request.Data.ResponderPeerID,
		Status:       status,
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
		return errors.New(fmt.Sprintf("Cannot send friends request to self"))
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
		log.Printf("Error writing/closing friends request stream to %s: %v", receiverPeerId, err)
		stream.Reset() // Reset on error
		return fmt.Errorf("failed to send/close friends request: %w", err)
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
		log.Printf("Error writing/closing friends response stream to %s: %v", receiverPeerId, err)
		stream.Reset() // Reset on error
		return fmt.Errorf("failed to send/close friends response: %w", err)
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
		return fmt.Errorf("failed to update friends relationship status: %w", err)
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

func (s *Service) GetFriendRequests() ([]types.FriendRelationship, error) {
	r, err := s.relationshipRepo.GetPendingRelations(s.ctx)

	if err != nil {
		log.Printf("Error getting friend requests: %v", err)
		return nil, err
	}

	return r, nil
}

// handleFriendResponsePollStream processes incoming friend response poll requests
func (s *Service) handleFriendResponsePollStream(stream network.Stream) {
	peerID := stream.Conn().RemotePeer()
	log.Printf("FriendResponsePoll: Received new stream from %s", peerID.ShortString())

	// Get the current relationship status
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch relationship status from repository
	relationship, err := s.relationshipRepo.GetRelationByPeerId(ctx, peerID.String())
	if err != nil {
		log.Printf("FriendResponsePoll Handler: Error fetching relationship for %s: %v", peerID.String(), err)
		stream.Reset()
		return
	}

	// Marshal the relationship to JSON
	responseBytes, err := json.Marshal(relationship)
	if err != nil {
		log.Printf("FriendResponsePoll Handler: Error marshaling relationship for %s: %v", peerID.String(), err)
		stream.Reset()
		return
	}

	// Write the response
	writer := bufio.NewWriter(stream)
	_, err = writer.Write(responseBytes)
	if err != nil {
		log.Printf("FriendResponsePoll Handler: Error writing response to %s: %v", peerID.String(), err)
		stream.Reset()
		return
	}

	err = writer.Flush()
	if err != nil {
		log.Printf("FriendResponsePoll Handler: Error flushing response to %s: %v", peerID.String(), err)
		stream.Reset()
		return
	}

	log.Printf("FriendResponsePoll Handler: Successfully responded to %s with status %s",
		peerID.String(), relationship.Status)

	stream.Close()
}

func (s *Service) PollForFriendResponse() {
	for {
		log.Printf("=============== Polling =================")
		pendingFriendRequests, err := s.GetFriendRequests()

		log.Printf("Pending friend requests: %v", pendingFriendRequests)

		if err != nil {
			log.Printf("Error polling for friend requests: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		for _, request := range pendingFriendRequests {
			if request.Status == types.FriendStatusSent {
				continue
			}

			log.Printf("Asking %s", request.PeerID)
			err := s.AskForFriendRequestResponse(request.PeerID)

			if err != nil {
				log.Printf("Error polling for friend requests: %v", err)
				continue
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func (s *Service) AskForFriendRequestResponse(peerId string) error {
	targetPID, err := peer.Decode(peerId)
	if err != nil {
		log.Printf("Invalid target PeerID format: %v", err)
		return fmt.Errorf("invalid peer ID format: %w", err)
	}

	// Check connectedness
	connectedness := (*s.appState.Node).Network().Connectedness(targetPID)
	if connectedness != network.Connected {
		log.Printf("Not connected to %s (State: %s). Attempting connection...", targetPID.ShortString(), connectedness)

		addrInfo := (*s.appState.Node).Peerstore().PeerInfo(targetPID)
		if len(addrInfo.Addrs) == 0 {
			log.Printf("No addresses found in Peerstore for %s. Cannot connect.", targetPID.ShortString())
			return fmt.Errorf("cannot connect to peer %s: no known addresses", targetPID.ShortString())
		}

		// Set up timeout for connection attempt
		connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer connectCancel()

		err = (*s.appState.Node).Connect(connectCtx, addrInfo)
		if err != nil {
			log.Printf("Failed to connect to %s: %v", targetPID.ShortString(), err)
			return fmt.Errorf("failed to establish connection with peer %s: %w", targetPID.ShortString(), err)
		}
		log.Printf("Successfully connected to %s", targetPID.ShortString())
	}

	// Open stream using the FriendResponsePollProtocolId
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("AskForFriendRequestResponse: Opening stream to %s for protocol %s", targetPID.ShortString(), core.FriendResponsePollProtocolId)
	stream, err := (*s.appState.Node).NewStream(ctx, targetPID, core.FriendResponsePollProtocolId)
	if err != nil {
		log.Printf("Failed to open stream to %s: %v", targetPID.ShortString(), err)
		return fmt.Errorf("failed to open stream to peer %s: %w", targetPID.ShortString(), err)
	}
	defer stream.Close()

	// Write our peer ID to the stream to identify ourselves
	myPeerID := (*s.appState.Node).ID().String()
	_, err = stream.Write([]byte(myPeerID))
	if err != nil {
		log.Printf("AskForFriendRequestResponse: Failed to write to stream: %v", err)
		stream.Reset()
		return fmt.Errorf("AskForFriendRequestResponse :failed to write to stream: %w", err)
	}

	// Read the response
	reader := bufio.NewReader(stream)
	responseBytes, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("AskForFriendRequestResponse: Failed to read from stream: %v", err)
		stream.Reset()
		return fmt.Errorf("AskForFriendRequestResponse :failed to read from stream: %w", err)
	}

	// If we got an empty response, the peer might not have processed our request yet
	if len(responseBytes) == 0 {
		return nil
	}

	// Parse the response
	var relationship types.FriendRelationship
	err = json.Unmarshal(responseBytes, &relationship)
	if err != nil {
		log.Printf("AskForFriendRequestResponse: Failed to unmarshal response: %v", err)
		return fmt.Errorf("AskForFriendRequestResponse: failed to parse response: %w", err)
	}

	if relationship.Status == types.FriendStatusPending {
		log.Printf("AskForFriendRequestResponse: Received friend request response with pending status from %s", targetPID.ShortString())
		return nil
	}

	s.bus.PublishAsync(events.FriendResponseReceivedEvent{
		SenderPeerId: targetPID.String(),
		Status:       relationship.Status,
		Timestamp:    relationship.ApprovedAt.String(),
	})
	log.Printf("AskForFriendRequestResponse: Received friend request response from %s: %s", targetPID.ShortString(), relationship.Status)
	return nil
}
