// src/services/websocket.js
const API_BASE_URL = import.meta.env.VITE_BACKEND_API_BASE_URL  || '127.0.0.1:59578';

class WebSocketService {
    constructor() {
        this.socket = null;
        this.messageCallbacks = [];
    }

    connect() {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            console.log('WebSocket already connected');
            return;
        }

        this.socket = new WebSocket('ws://' + API_BASE_URL + '/ws');

        this.socket.onopen = () => {
            console.log('WebSocket connection established');
        };

        this.socket.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.messageCallbacks.forEach(callback => callback(data));
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };

        this.socket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.socket.onclose = (event) => {
            console.log('WebSocket connection closed:', event.code, event.reason);
            // Try to reconnect after a delay
            setTimeout(() => this.connect(), 5000);
        };
    }

    sendMessage(targetPeerId, message) { // for Direct Messages
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            console.error('WebSocket not connected');
            return false;
        }

        const payload = {
            type: 'DIRECT_MESSAGE',
            payload: {
                target_peer_id: targetPeerId,
                message: message
            }
        };

        this.socket.send(JSON.stringify(payload));
        return true;
    }

    sendGroupMessage(groupId, message) { // for Group Messages
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            console.error('WebSocket not connected');
            return false;
        }

        const payload = {
            type: 'GROUP_MESSAGE',
            payload: {
                group_id: groupId,
                message: message
            }
        };

        this.socket.send(JSON.stringify(payload));
        return true;
    }


    addMessageListener(callback) {
        this.messageCallbacks.push(callback);
        return () => {
            this.messageCallbacks = this.messageCallbacks.filter(cb => cb !== callback);
        };
    }

    disconnect() {
        if (this.socket) {
            this.socket.close();
            this.socket = null;
        }
    }
}

// Create a singleton instance
const websocketService = new WebSocketService();
export default websocketService;