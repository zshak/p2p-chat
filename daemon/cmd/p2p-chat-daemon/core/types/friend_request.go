package types

type FriendRequestData struct {
	SenderPeerID string `json:"requester_id"`
	Timestamp    string `json:"timestamp"`
}

type FriendRequest struct {
	Data            FriendRequestData `json:"data"`
	SenderSignature []byte            `json:"signature"`
}
