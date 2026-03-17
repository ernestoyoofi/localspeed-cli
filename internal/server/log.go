package server

import (
	"log"
	"net/http"
	"strings"
)

// LogAction logs server activities in the required format.
func LogAction(ip, action, reqType, size string) {
	log.Printf("From %s - Action: %s - Type: %s - Size: %s", ip, action, reqType, size)
}

// GetIP helper function to extract IP from request
func GetIP(r *http.Request) string {
	remoteAddr := r.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}
