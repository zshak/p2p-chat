package types

type FriendResponseData struct {
	ResponderPeerID string `json:"responder_id"`
	IsApproved      bool   `json:"is_approved"`
	Timestamp       string `json:"timestamp"`
}

type FriendResponse struct {
	Data            FriendResponseData `json:"data"`
	SenderSignature []byte             `json:"signature"`
}
