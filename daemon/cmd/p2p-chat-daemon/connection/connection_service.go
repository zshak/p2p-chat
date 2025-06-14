package connection

import (
	"context"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
	"sync"
	"time"

	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"

	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	friendStatusCheckInterval = 15 * time.Second // How often to ping friends
	pingTimeout               = 15 * time.Second // Timeout for a single ping
)

// Service periodically checks the online status of friends.
type Service struct {
	ctx             context.Context
	cancelFunc      context.CancelFunc
	appState        *core.AppState
	friendRepo      storage.RelationshipRepository
	eventBus        *bus.EventBus
	wg              sync.WaitGroup
	lastKnownStatus map[peer.ID]bool
	statusMutex     sync.RWMutex
	pingService     *ping.PingService
}

// NewConnectionService creates a new ConnectionService.
func NewConnectionService(
	parentCtx context.Context,
	app *core.AppState,
	repo storage.RelationshipRepository,
	eb *bus.EventBus,
) *Service {
	ctx, cancel := context.WithCancel(parentCtx)
	return &Service{
		ctx:             ctx,
		cancelFunc:      cancel,
		appState:        app,
		friendRepo:      repo,
		eventBus:        eb,
		lastKnownStatus: make(map[peer.ID]bool),
	}
}

// Start launches the background goroutine for checking friend statuses.
func (s *Service) Start() {
	ps := ping.NewPingService(*s.appState.Node)
	s.pingService = ps

	log.Println("Connection Service: Starting friend online status checker...")
	s.wg.Add(1)
	go s.statusCheckLoop()
}

// Stop signals the background goroutine to stop and waits for it.
func (s *Service) Stop() {
	log.Println("Connection Service: Stopping...")
	s.cancelFunc()
	s.wg.Wait()
	log.Println("Connection Service: Stopped.")
}

func (s *Service) IsOnline(id peer.ID) bool {
	s.statusMutex.RLock()
	defer s.statusMutex.RUnlock()

	isOnline, known := s.lastKnownStatus[id]
	if !known {
		return false
	}
	return isOnline
}

func (s *Service) statusCheckLoop() {
	defer s.wg.Done()
	log.Println("Connection Service: Status check loop initiated.")
	s.checkAllFriends()

	ticker := time.NewTicker(friendStatusCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Connection Service: Status check loop stopping due to context cancellation.")
			return
		case <-ticker.C:
			log.Println("Connection Service: Performing periodic friend status check...")
			s.checkAllFriends()
		}
	}
}

func (s *Service) checkAllFriends() {
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dbCancel()
	approvedFriends, err := s.friendRepo.GetAcceptedRelations(dbCtx)
	if err != nil {
		log.Printf("Connection Service: Error fetching approved friends: %v", err)
		return
	}

	if len(approvedFriends) == 0 {
		return
	}

	log.Printf("Connection Service: Checking status of %d approved friend(s)...", len(approvedFriends))
	var checkWg sync.WaitGroup
	for _, friendRel := range approvedFriends {
		checkWg.Add(1)
		go func(fRel types.FriendRelationship) {
			defer checkWg.Done()
			if s.ctx.Err() != nil {
				return
			}

			friendPID, err := peer.Decode(fRel.PeerID)
			if err != nil {
				log.Printf("Connection Service: Error decoding friend PeerID %s: %v", fRel.PeerID, err)
				return
			}

			// Do not ping self
			if friendPID == (*s.appState.Node).ID() {
				log.Printf("Yo i am pingin myself wth")
				return
			}

			s.pingPeerAndNotify(friendPID)
		}(friendRel)
	}
	checkWg.Wait()
	log.Println("Connection Service: Finished checking all friends in this round.")
}

func (s *Service) pingPeerAndNotify(targetPeerID peer.ID) {
	pingRpcCtx, pingRpcCancel := context.WithTimeout(s.ctx, pingTimeout)
	defer pingRpcCancel()

	res := <-s.pingService.Ping(pingRpcCtx, targetPeerID)

	var isOnline bool
	var rtt time.Duration

	if res.Error == nil {
		isOnline = true
		rtt = res.RTT
		log.Printf("Connection Service: Ping to %s SUCCESS, RTT: %s", targetPeerID.ShortString(), rtt)
	} else {
		isOnline = false
		log.Printf("Connection Service: Ping to %s FAILED: %v", targetPeerID.ShortString(), res.Error)
	}

	s.updateAndNotifyStatus(targetPeerID, isOnline, rtt)
}

// updateAndNotifyStatus updates the known status and publishes an event if changed.
func (s *Service) updateAndNotifyStatus(peerID peer.ID, isOnline bool, rtt time.Duration) {
	s.statusMutex.Lock()
	lastStatus, known := s.lastKnownStatus[peerID]

	if !known || lastStatus != isOnline {
		s.lastKnownStatus[peerID] = isOnline
		s.statusMutex.Unlock()

		log.Printf("Connection Service: Status CHANGE for %s: Online = %t (RTT: %s)", peerID.ShortString(), isOnline, rtt)
		s.eventBus.PublishAsync(events.FriendOnlineStatusChangedEvent{
			PeerID:   peerID.String(),
			IsOnline: isOnline,
			LastSeen: time.Now(),
			RTT:      rtt,
		})
	} else {
		s.statusMutex.Unlock() // No change
	}
}
