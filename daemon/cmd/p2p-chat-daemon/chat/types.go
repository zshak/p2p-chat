package chat

import "time"

type GroupChatRequest struct {
	MemberPeers []string
	Key         []byte
	Name        string
	Id          string
}

type GroupChatMessages struct {
	Messages []GroupChatMessage
}

type GroupChatMessage struct {
	SenderPeerId string
	Message      string
	Time         time.Time
}

type Messages struct {
	Messages []Message
}

type Message struct {
	SendTime   time.Time
	Message    string
	IsOutgoing bool
}
