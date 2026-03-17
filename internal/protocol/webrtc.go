package protocol

const (
	DataChannelName = "localspeed-dc"
)

// PingMessage is used on the WebRTC DataChannel to measure Jitter and Packet Loss.
type PingMessage struct {
	Seq       int   `json:"seq"`
	Timestamp int64 `json:"timestamp"`
}
