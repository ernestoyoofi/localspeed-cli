package client

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ernestoyoofi/localspeed-cli/internal/protocol"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type LatencyStats struct {
	count     int
	latencies []float64
	min       float64
	max       float64
	sum       float64
	jitterSum float64
	mu        sync.Mutex
}

func (s *LatencyStats) Add(lat float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count == 0 {
		s.min = lat
		s.max = lat
	} else {
		if lat < s.min {
			s.min = lat
		}
		if lat > s.max {
			s.max = lat
		}
		s.jitterSum += math.Abs(lat - s.latencies[len(s.latencies)-1])
	}
	s.latencies = append(s.latencies, lat)
	s.sum += lat
	s.count++
}

func (s *LatencyStats) Avg() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count == 0 {
		return 0
	}
	return s.sum / float64(s.count)
}

func (s *LatencyStats) Jitter() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count < 2 {
		return 0
	}
	return s.jitterSum / float64(s.count-1)
}

func (s *LatencyStats) Summary() (avg, jitter, low, high float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count == 0 {
		return 0, 0, 0, 0
	}
	avg = s.sum / float64(s.count)
	if s.count > 1 {
		jitter = s.jitterSum / float64(s.count-1)
	}
	low = s.min
	high = s.max
	return
}

func RunBenchmark(targetURL, sampleStr string, insecure bool, savePath string) {
	u, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Invalid target URL: %v", err)
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/test/ws", wsScheme, u.Host)

	tlsConfig := &tls.Config{InsecureSkipVerify: insecure}
	dialer := websocket.Dialer{TLSClientConfig: tlsConfig}

	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Failed to connect to signaling server: %v", err)
	}
	defer conn.Close()

	serverName := u.Host
	if resp != nil && resp.Header.Get("X-Speedtest-Name") != "" {
		serverName = resp.Header.Get("X-Speedtest-Name")
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Failed to create PeerConnection: %v", err)
	}
	defer pc.Close()

	ordered := false
	dcInit := &webrtc.DataChannelInit{Ordered: &ordered}
	dc, err := pc.CreateDataChannel(protocol.DataChannelName, dcInit)
	if err != nil {
		log.Fatalf("Failed to create DataChannel: %v", err)
	}

	dcReady := make(chan struct{})
	dc.OnOpen(func() { close(dcReady) })

	var currentStats *LatencyStats
	var statsMu sync.Mutex
	var pingsSent int64
	var pingsReceived int64

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var pingMsg protocol.PingMessage
		if err := json.Unmarshal(msg.Data, &pingMsg); err == nil {
			now := time.Now().UnixNano()
			rttMs := float64(now-pingMsg.Timestamp) / 1e6

			atomic.AddInt64(&pingsReceived, 1)

			statsMu.Lock()
			st := currentStats
			statsMu.Unlock()

			if st != nil {
				st.Add(rttMs)
			}
		}
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		data, _ := json.Marshal(c.ToJSON())
		b, _ := json.Marshal(protocol.WsMessage{Event: "candidate", Data: string(data)})
		conn.WriteMessage(websocket.TextMessage, b)
	})

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var wsMsg protocol.WsMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				continue
			}
			switch wsMsg.Event {
			case "answer":
				var answer webrtc.SessionDescription
				json.Unmarshal([]byte(wsMsg.Data), &answer)
				pc.SetRemoteDescription(answer)
			case "candidate":
				var candidate webrtc.ICECandidateInit
				json.Unmarshal([]byte(wsMsg.Data), &candidate)
				pc.AddICECandidate(candidate)
			}
		}
	}()

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Fatalf("CreateOffer err: %v", err)
	}
	pc.SetLocalDescription(offer)
	offerData, _ := json.Marshal(offer)
	b, _ := json.Marshal(protocol.WsMessage{Event: "offer", Data: string(offerData)})
	conn.WriteMessage(websocket.TextMessage, b)

	fmt.Print("Waiting for WebRTC connection...")
	select {
	case <-dcReady:
		fmt.Print("\r\033[K") // Clear waiting message
	case <-time.After(10 * time.Second):
		log.Fatalf("\nTimeout waiting for WebRTC DataChannel to open")
	}

	sizeBytes := parseSize(sampleStr)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	csvWriter, err := NewCSVWriter(savePath)
	if err != nil {
		log.Printf("Warning: Failed to create CSV file: %v. Continuing without saving.\n", err)
	} else if csvWriter != nil {
		defer csvWriter.Close()
	}

	// -----------------------------
	// Phase 1: Info Server
	// -----------------------------
	fmt.Println("\033[1m   [ Bechmark Information ]\x1b[0m")
	fmt.Printf("      Server: %s\n", serverName)
	fmt.Printf("        Host: %s\n", u.Host)

	// -----------------------------
	// Phase 2: Idle Latency
	// -----------------------------
	statsMu.Lock()
	idleStats := &LatencyStats{}
	currentStats = idleStats
	statsMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Pinger
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond) // Fast ping
		defer ticker.Stop()
		seq := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				atomic.AddInt64(&pingsSent, 1)
				msg := protocol.PingMessage{Seq: seq, Timestamp: time.Now().UnixNano()}
				b, _ := json.Marshal(msg)
				dc.Send(b)
				seq++
			}
		}
	}()

	time.Sleep(2 * time.Second)
	idleAvg, idleJitter, idleLow, idleHigh := idleStats.Summary()
	PrintIdleLatency(idleAvg, idleJitter, idleLow, idleHigh)

	var dlBytes int64
	var ulBytes int64

	// CSV Monitor Loop
	if csvWriter != nil {
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			lastDl := int64(0)
			lastUl := int64(0)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					dl := atomic.LoadInt64(&dlBytes)
					ul := atomic.LoadInt64(&ulBytes)

					dlDelta := dl - lastDl
					ulDelta := ul - lastUl
					lastDl = dl
					lastUl = ul

					dlSpeed := float64(dlDelta) * 2 / 1024 // KBps
					ulSpeed := float64(ulDelta) * 2 / 1024 // KBps

					statsMu.Lock()
					st := currentStats
					statsMu.Unlock()

					jitter := float64(0)
					if st != nil {
						jitter = st.Jitter()
					}

					dlProg := float64(dl) / float64(sizeBytes) * 100
					ulProg := float64(ul) / float64(sizeBytes) * 100

					if dlDelta > 0 {
						csvWriter.WriteRow(ResultRow{Type: "Download", SpeedKBps: dlSpeed, Progress: dlProg, JitterMs: jitter, PacketLoss: 0})
					}
					if ulDelta > 0 {
						csvWriter.WriteRow(ResultRow{Type: "Upload", SpeedKBps: ulSpeed, Progress: ulProg, JitterMs: jitter, PacketLoss: 0})
					}
				}
			}
		}()
	}

	// -----------------------------
	// Phase 2: Download
	// -----------------------------
	statsMu.Lock()
	dlStats := &LatencyStats{}
	currentStats = dlStats
	statsMu.Unlock()

	downloadURL := fmt.Sprintf("%s/test/download?size=%s", targetURL, sampleStr)
	dlReq, _ := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)

	startDl := time.Now()
	dlDone := make(chan bool)
	go func() {
		resp, err := httpClient.Do(dlReq)
		if err == nil {
			defer resp.Body.Close()
			buf := make([]byte, 64*1024)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					atomic.AddInt64(&dlBytes, int64(n))
				}
				if err != nil {
					break
				}
			}
		}
		dlDone <- true
	}()

	dlRunning := true
	for dlRunning {
		select {
		case <-dlDone:
			dlRunning = false
		default:
			currentDl := atomic.LoadInt64(&dlBytes)
			elapsed := time.Since(startDl).Seconds()
			if elapsed > 0 {
				speedMbps := (float64(currentDl) / elapsed) * 8 / 1e6
				prog := (float64(currentDl) / float64(sizeBytes)) * 100
				PrintProgressLine("Download", speedMbps, prog, dlStats.Avg())
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	dlDuration := time.Since(startDl).Seconds()
	finalDlBytes := atomic.LoadInt64(&dlBytes)
	finalDlSpeed := (float64(finalDlBytes) / dlDuration) * 8 / 1e6
	dlAvg, dlJitter, dlLow, dlHigh := dlStats.Summary()
	// Clear the progress line before printing final phase results
	fmt.Print("\r\033[K")
	PrintPhaseFinished("Download", finalDlSpeed, finalDlBytes, dlAvg, dlJitter, dlLow, dlHigh)

	// -----------------------------
	// Phase 3: Upload
	// -----------------------------
	statsMu.Lock()
	ulStats := &LatencyStats{}
	currentStats = ulStats
	statsMu.Unlock()

	uploadURL := fmt.Sprintf("%s/test/upload?size=%s", targetURL, sampleStr)
	ulReq, _ := http.NewRequestWithContext(ctx, "POST", uploadURL, io.LimitReader(randReader{}, sizeBytes))
	ulReq.Header.Set("Content-Type", "application/octet-stream")
	ulReq.Header.Set("X-File-Size", sampleStr)

	pr, pw := io.Pipe()
	go func() {
		buf := make([]byte, 64*1024)
		rand.Read(buf)
		written := int64(0)
		for written < sizeBytes {
			toWrite := int64(len(buf))
			if sizeBytes-written < toWrite {
				toWrite = sizeBytes - written
			}
			n, err := pw.Write(buf[:toWrite])
			if err != nil {
				break
			}
			atomic.AddInt64(&ulBytes, int64(n))
			written += int64(n)
		}
		pw.Close()
	}()
	ulReq.Body = io.NopCloser(pr)

	startUl := time.Now()
	ulDone := make(chan bool)
	go func() {
		resp, err := httpClient.Do(ulReq)
		if err == nil {
			resp.Body.Close()
		}
		ulDone <- true
	}()

	ulRunning := true
	for ulRunning {
		select {
		case <-ulDone:
			ulRunning = false
		default:
			currentUl := atomic.LoadInt64(&ulBytes)
			elapsed := time.Since(startUl).Seconds()
			if elapsed > 0 {
				speedMbps := (float64(currentUl) / elapsed) * 8 / 1e6
				prog := (float64(currentUl) / float64(sizeBytes)) * 100
				PrintProgressLine("Upload", speedMbps, prog, ulStats.Avg())
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	ulDuration := time.Since(startUl).Seconds()
	finalUlBytes := atomic.LoadInt64(&ulBytes)
	finalUlSpeed := (float64(finalUlBytes) / ulDuration) * 8 / 1e6
	ulAvg, ulJitter, ulLow, ulHigh := ulStats.Summary()
	// Clear the progress line before printing final phase results
	fmt.Print("\r\033[K")
	PrintPhaseFinished("Upload", finalUlSpeed, finalUlBytes, ulAvg, ulJitter, ulLow, ulHigh)

	// Wrap up
	cancel() // Stop WebRTC pinger and CSV monitor

	sent := atomic.LoadInt64(&pingsSent)
	recv := atomic.LoadInt64(&pingsReceived)
	loss := float64(-1)
	if sent > 0 {
		loss = float64(sent-recv) / float64(sent) * 100
	}
	if loss < 0 {
		loss = 0
	}

	if csvWriter != nil {
		csvWriter.WriteRow(ResultRow{Type: "Result_Download_Avg", SpeedKBps: finalDlSpeed * 1024 / 8, Progress: 100, JitterMs: dlJitter, PacketLoss: loss})
		csvWriter.WriteRow(ResultRow{Type: "Result_Upload_Avg", SpeedKBps: finalUlSpeed * 1024 / 8, Progress: 100, JitterMs: ulJitter, PacketLoss: loss})
	}

	PrintFinalResult(targetURL,
		idleAvg, idleJitter, idleLow, idleHigh,
		finalDlSpeed, float64(finalDlBytes)/1024/1024, dlAvg, dlJitter, dlLow, dlHigh,
		finalUlSpeed, float64(finalUlBytes)/1024/1024, ulAvg, ulJitter, ulLow, ulHigh,
		loss)
}

func parseSize(sizeStr string) int64 {
	sizeStr = strings.ToUpper(sizeStr)
	switch sizeStr {
	case "512KB":
		return 512 * 1024
	case "1MB":
		return 1 * 1024 * 1024
	case "5MB":
		return 5 * 1024 * 1024
	case "10MB":
		return 10 * 1024 * 1024
	case "20MB":
		return 20 * 1024 * 1024
	case "50MB":
		return 50 * 1024 * 1024
	case "100MB":
		return 100 * 1024 * 1024
	case "200MB":
		return 200 * 1024 * 1024
	case "500MB":
		return 500 * 1024 * 1024
	case "1GB":
		return 1 * 1024 * 1024 * 1024
	case "10GB":
		return 10 * 1024 * 1024 * 1024
	default:
		return 10 * 1024 * 1024 // Default 10MB
	}
}

type randReader struct{}

func (randReader) Read(p []byte) (n int, err error) {
	return rand.Read(p)
}
