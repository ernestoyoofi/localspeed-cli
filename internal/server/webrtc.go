package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ernestoyoofi/localspeed-cli/internal/protocol"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func WsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, http.Header{"X-Speedtest-Name": []string{GetServerName()}})
	if err != nil {
		log.Print("upgrade error:", err)
		return
	}
	defer c.Close()

	ip := GetIP(r)
	LogAction(ip, "START", "WEBRTC", "N/A")
	defer LogAction(ip, "FINISH", "WEBRTC", "N/A")

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Println("NewPeerConnection err:", err)
		return
	}
	defer peerConnection.Close()

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			// Echo the ping back to client for RTT calculation
			d.Send(msg.Data)
		})
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		data, _ := json.Marshal(candidate.ToJSON())
		sendWSMsg(c, protocol.WsMessage{Event: "candidate", Data: string(data)})
	})

	// Read messages from WebSocket
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg protocol.WsMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			continue
		}

		switch wsMsg.Event {
		case "offer":
			var offer webrtc.SessionDescription
			if err := json.Unmarshal([]byte(wsMsg.Data), &offer); err != nil {
				continue
			}
			if err := peerConnection.SetRemoteDescription(offer); err != nil {
				continue
			}
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				continue
			}
			if err := peerConnection.SetLocalDescription(answer); err != nil {
				continue
			}
			data, _ := json.Marshal(answer)
			sendWSMsg(c, protocol.WsMessage{Event: "answer", Data: string(data)})
		case "candidate":
			var candidate webrtc.ICECandidateInit
			if err := json.Unmarshal([]byte(wsMsg.Data), &candidate); err != nil {
				continue
			}
			if err := peerConnection.AddICECandidate(candidate); err != nil {
				continue
			}
		}
	}
}

func sendWSMsg(c *websocket.Conn, wsMsg protocol.WsMessage) error {
	b, _ := json.Marshal(wsMsg)
	return c.WriteMessage(websocket.TextMessage, b)
}
