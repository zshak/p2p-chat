package api

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
)

const (
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.RemoteAddr
		println(origin)
		if strings.HasPrefix(origin, "http://localhost") ||
			strings.HasPrefix(origin, "http://127.0.0.1") ||
			strings.HasPrefix(origin, "127.0.0.1") {
			return true
		}
		return false
	},
}

// handleWebSocket is the HTTP handler that upgrades connections to socket
func (h *ApiHandler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("API WS Handler: Received HTTP request for upgrade...")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("API WS Handler: Failed to upgrade: %v", err)
		return
	}

	h.wsMu.Lock()
	if h.wsConn != nil {
		h.wsConn.Close()
		h.wsConn = conn
	} else {
		h.wsConn = conn
	}
	h.wsMu.Unlock()

	remoteAddr := conn.RemoteAddr()
	log.Printf("API WS Handler: Connection established from %s", remoteAddr)

	defer func() {
		log.Printf("API WS Handler: Closing connection from %s", remoteAddr)
		conn.Close()
	}()

	h.readLoop(conn)

	log.Printf("API WS Handler: readLoop finished for %s", remoteAddr)
}

// readLoop handles reading messages from a single WebSocket connection
func (h *ApiHandler) readLoop(conn *websocket.Conn) {
	remoteAddr := conn.RemoteAddr()
	defer log.Printf("API WS ReadLoop: Exiting for %s", remoteAddr)

	conn.SetReadLimit(maxMessageSize)

	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("API WS ReadLoop: Unexpected close error for %s: %v", remoteAddr, err)
			} else {
				log.Printf("API WS ReadLoop: WebSocket closed or read error for %s: %v", remoteAddr, err)
			}
			break
		}

		var msg MessageRequest
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("API WS ReadLoop: can not deserialize mesage %s", string(messageBytes))
			continue
		}

		if msg.Type == WsMsgTypeDirectMessage {
			var req WsDirectMessageRequestPayload
			json.Unmarshal(msg.Payload, &req)

			h.chatService.SendMessage(req.TargetPeerID, req.Message)
		} else if msg.Type == WsMsgTypeGroupMessage {
			var req WsGroupMessageRequestPayload
			json.Unmarshal(msg.Payload, &req)

			h.chatService.SendGroupMessage(req.GroupId, req.Message)
		}
	}
}

func (h *ApiHandler) send(msg []byte) {
	h.wsMu.Lock()
	defer h.wsMu.Unlock()

	if h.wsConn == nil {
		log.Printf("API WS Send: No connection to send message")
		return
	}

	err := h.wsConn.WriteMessage(websocket.TextMessage, msg)

	if err != nil {
		log.Printf("API WS Send: Failed to send message: %v", err)
		return
	}

	log.Printf("API WS Send: message sent %s", msg)
}
