package client

import (
	"fmt"
	"strings"
)

// PrintIdleLatency prints the initial idle latency row
func PrintIdleLatency(latency, jitter, low, high float64) {
	fmt.Printf("Idle Latency:  %8.2f ms  (jitter: %.2fms, low: %.2fms, high: %.2fms)\n", latency, jitter, low, high)
}

var spinnerFrames = []string{"|", "/", "-", "\\"}
var spinnerIdx = 0

// PrintProgressLine prints an in-place updating progress bar identical to Ookla CLI.
func PrintProgressLine(prefix string, speedMbps float64, progress float64, latency float64) {
	// e.g. [==/                 ]
	bars := int(progress / 5) // maximum 20 pieces
	if bars > 20 {
		bars = 20
	}
	
	spinnerChar := spinnerFrames[spinnerIdx%len(spinnerFrames)]
	spinnerIdx++

	barStr := ""
	if bars >= 20 {
		barStr = strings.Repeat("=", 20)
	} else {
		barStr = strings.Repeat("=", bars) + spinnerChar + strings.Repeat(" ", 20-bars-1)
	}
	
    // We use \r to overwrite line
	fmt.Printf("\r    %8s: %8.2f Mbps [%s] %3.0f%%   - latency: %.2f ms", prefix, speedMbps, barStr, progress, latency)
}

// PrintPhaseFinished clears the progress line and prints the final stats for the completed testing phase.
func PrintPhaseFinished(prefix string, speedMbps float64, bytesUsed int64, lat, jitter, low, high float64) {
	mbUsed := float64(bytesUsed) / 1024 / 1024
	fmt.Printf("    %8s: %8.2f Mbps (data used: %.1f MB)\n", prefix, speedMbps, mbUsed)
	fmt.Printf("               %8.2f ms  (jitter: %.2fms, low: %.2fms, high: %.2fms)\n", lat, jitter, low, high)
}

// PrintFinalResult outputs the completion table matching Ookla formats.
func PrintFinalResult(targetURL string,
	idleLat, idleJitter, idleLow, idleHigh float64,
	dlSpeed, dlData, dlLat, dlJitter, dlLow, dlHigh float64,
	ulSpeed, ulData, ulLat, ulJitter, ulLow, ulHigh float64,
	packetLoss float64) {
	
	if packetLoss >= 0 {
		fmt.Printf(" Packet Loss: %.2f %%\n", packetLoss)
	} else {
		fmt.Println(" Packet Loss: Not available.")
	}

	fmt.Println("\n\033[1m   [ Quality Estimates ]\x1b[0m")

	latencyMs := idleLat
	jitterMs := idleJitter
	bandwidthMbps := (dlSpeed + ulSpeed) / 2

	// Gaming: Excellent if Latency < 20ms, Jitter < 5ms.
	gaming := "\x1b[31mPoor\x1b[0m"
	if latencyMs < 20 && jitterMs < 5 {
		gaming = "\x1b[32mExcellent\x1b[0m"
	} else if latencyMs < 50 && jitterMs < 10 {
		gaming = "\x1b[33mGood\x1b[0m"
	}
	fmt.Printf("      Gaming: %s\n", gaming)

	// Streaming: Excellent if Bandwidth > 25Mbps stabil.
	streaming := "\x1b[31mPoor\x1b[0m"
	if bandwidthMbps > 25 {
		streaming = "\x1b[32mExcellent\x1b[0m"
	} else if bandwidthMbps > 5 {
		streaming = "\x1b[33mGood\x1b[0m"
	}
	fmt.Printf("   Streaming: %s\n", streaming)

	// Browsing: Excellent if Packet Loss 0%.
	browsing := "\x1b[31mPoor\x1b[0m"
	if packetLoss == 0 {
		browsing = "\x1b[32mExcellent\x1b[0m"
	} else if packetLoss < 2 {
		browsing = "\x1b[33mGood\x1b[0m"
	}
	fmt.Printf("    Browsing: %s\n\n", browsing)
}
