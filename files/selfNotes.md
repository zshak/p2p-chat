### To run:
1. cd /daemon/cmd/p2p-chat-daemon
2. ./p2p-chat-daemon -api 127.0.0.1:59579 -pub -mdns -key key1 -db chat1.db

## Decision Matrix: WebSocket vs REST API

| Feature | Use WebSocket | Use REST API |
|---------|--------------|--------------|
| Send message | ✅ | ❌ |
| Receive messages | ✅ | ❌ |
| User login | ❌ | ✅ |
| User registration | ❌ | ✅ |
| Upload files | ❌ | ✅ |
| Message history | ❌ | ✅ |
| Create room | ❌ | ✅ |
| Typing indicator | ✅ | ❌ |
| Online status | ✅ | ❌ |
| Update profile | ❌ | ✅ |
