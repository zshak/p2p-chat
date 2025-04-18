package p2p

import (
	"bufio"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core" // For protocol ID
	"strings"
)

// ChatService handles registration and processing of chat protocol streams.
type ChatService struct {
	host host.Host
}

// NewChatService creates a new chat service instance.
func NewChatService(h host.Host) *ChatService {
	if h == nil {
		return nil
	} // Or return error
	return &ChatService{host: h}
}

// RegisterHandler sets the stream handler for the chat protocol.
func (cs *ChatService) RegisterHandler() {
	if cs.host == nil {
		log.Println("P2P Chat Service: ERROR - Cannot register handler, host is nil.")
		return
	}
	log.Printf("P2P Chat Service: Registering handler for protocol %s", core.ChatProtocolID)
	cs.host.SetStreamHandler(core.ChatProtocolID, cs.handleStream)
}

// handleStream is the internal handler function.
func (cs *ChatService) handleStream(stream network.Stream) {
	peerID := stream.Conn().RemotePeer()
	log.Printf("Chat Handler: Received new stream from %s", peerID.ShortString())
	defer stream.Close()

	reader := bufio.NewReader(stream)
	message, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Chat Handler: Error reading from stream from %s: %v", peerID.ShortString(), err)
		stream.Reset()
		return
	}
	message = strings.TrimSpace(message)
	log.Printf("Chat Handler: Received message from %s: <<< %s >>>", peerID.ShortString(), message)
	// TODO: Pass message to application UI event queue
}
