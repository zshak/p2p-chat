package chat

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"strings"
	"time"
)

// Service manages the chat protocol
type Service struct {
	appState *core.AppState
}

// NewProtocolHandler creates a new chat protocol handler
func NewProtocolHandler(app *core.AppState) *Service {
	return &Service{
		appState: app,
	}
}

// Register registers the chat protocol handler with the node
func (h *Service) Register() {
	log.Printf("Registering chat protocol handler (%s)...", core.ChatProtocolID)
	(*h.appState.Node).SetStreamHandler(core.ChatProtocolID, h.handleChatStream)
}

// handleChatStream processes incoming chat streams
func (h *Service) handleChatStream(stream network.Stream) {
	peerID := stream.Conn().RemotePeer()
	log.Printf("Chat: Received new stream from %s", peerID.ShortString())

	// Use a buffered reader for efficiency
	reader := bufio.NewReader(stream)

	// Read the message (assuming one message per stream, ending with newline for this simple example)
	message, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Chat: Error reading from stream from %s: %v", peerID.ShortString(), err)
		stream.Reset() // Abruptly close the stream on error
		return
	}

	// Trim trailing newline
	message = strings.TrimSpace(message)

	// Log the received message (replace with actual message handling later)
	log.Printf("Chat: Received message from %s: <<< %s >>>", peerID.ShortString(), message)

	// For this simple test, we can just close the stream after reading.
	// Alternatively, the sender could close it.
	stream.Close()
}

// SendMessage sends a chat message to a peer
func (h *Service) SendMessage(targetPeerId string, message string) error {
	// Parse PeerID
	targetPID, err := peer.Decode(targetPeerId)
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid target PeerID format: %v", err))
	}

	if h.appState.State != core.StateRunning || h.appState.Node == nil {
		return errors.New(fmt.Sprintf(fmt.Sprintf("Node is not ready (state: %s)", h.appState.State)))
	}

	// Don't send to self
	if targetPID == (*h.appState.Node).ID() {
		return errors.New(fmt.Sprintf("Cannot send chat message to self"))
	}

	log.Printf("Chat API: Checking connectedness to %s", targetPID.ShortString())
	connectedness := (*h.appState.Node).Network().Connectedness(targetPID)

	if connectedness != network.Connected {
		log.Printf("Chat API: Not connected to %s (State: %s). Attempting connection...", targetPID.ShortString(), connectedness)

		// Need AddrInfo to connect. Get latest from Peerstore.
		// Discovery should be populating this periodically.
		addrInfo := (*h.appState.Node).Peerstore().PeerInfo(targetPID)
		if len(addrInfo.Addrs) == 0 {
			log.Printf("Chat API: No addresses found in Peerstore for %s. Cannot connect.", targetPID.ShortString())
			return errors.New(fmt.Sprintf("Cannot connect to peer %s: No known addresses", targetPID.ShortString()))
		}

		// Use a separate context and timeout for the connection attempt
		connectCtx, connectCancel := context.WithTimeout(context.Background(), 60*time.Second) // Longer timeout for connect
		defer connectCancel()

		err = (*h.appState.Node).Connect(connectCtx, addrInfo) // Use the AddrInfo from Peerstore
		if err != nil {
			log.Printf("Chat API: Failed to connect to %s: %v", targetPID.ShortString(), err)
			return errors.New(fmt.Sprintf("Failed to establish connection with peer %s: %v", targetPID.ShortString(), err))
		}
		log.Printf("Chat API: Successfully connected to %s.", targetPID.ShortString())
		// Now we expect connectedness == network.Connected
	} else {
		log.Printf("Chat API: Already connected to %s.", targetPID.ShortString())
	}

	// --- Open Stream to Peer ---
	log.Printf("Chat API: Attempting to open stream to %s for protocol %s", targetPID.ShortString(), core.ChatProtocolID)

	// Use a timeout for stream opening
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ctx = network.WithAllowLimitedConn(ctx, "mito")
	defer cancel()

	stream, err := (*h.appState.Node).NewStream(ctx, targetPID, core.ChatProtocolID)
	if err != nil {
		log.Printf("Chat API: Failed to open stream to %s: %v", targetPID.ShortString(), err)
		return errors.New(fmt.Sprintf("Failed to connect/open stream to peer %s: %v", targetPID.ShortString(), err))
	}
	log.Printf("Chat API: Stream opened successfully to %s", targetPID.ShortString())

	// --- Send Message ---
	writer := bufio.NewWriter(stream)
	_, err = writer.WriteString(message + "\n") // Add newline delimiter
	if err == nil {
		err = writer.Flush() // Ensure data is sent
	}

	if err != nil {
		log.Printf("Chat API: Failed to write message to %s: %v", targetPID.ShortString(), err)
		stream.Reset()
		return errors.New(fmt.Sprintf("Failed to send message: %v", err))
	}

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
