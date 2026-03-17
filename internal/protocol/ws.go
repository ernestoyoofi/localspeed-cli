package protocol

// WsMessage defines the structure of WebSocket signaling messages between client and server.
type WsMessage struct {
	Event string `json:"event"` // e.g., "offer", "answer", "candidate", "start", "error", "ready"
	Data  string `json:"data"`  // JSON stringified payload (SDP or ICE Candidate)
}
