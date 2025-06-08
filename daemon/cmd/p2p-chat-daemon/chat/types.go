package chat

type GroupChatRequest struct {
	MemberPeers []string
	Key         []byte
	Name        string
	Id          string
}
