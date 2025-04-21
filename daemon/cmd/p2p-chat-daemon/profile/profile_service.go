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
	"time"
)

type Service struct {
	appState *core.AppState
	bus      *bus.EventBus
}

// NewProtocolHandler creates a new chat protocol handler
func NewProtocolHandler(app *core.AppState, bus *bus.EventBus) *Service {
	return &Service{
		appState: app,
		bus:      bus,
	}
}

func (s *Service) Register() {
	log.Printf("Registering Friend Request protocol handler (%s)...", core.FriendRequestProtocolID)
	(*s.appState.Node).SetStreamHandler(core.FriendRequestProtocolID, s.handleFriendRequestStream)
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

	log.Printf("Friend request sent successfully to %s", receiverPeerId)
	return nil
}
