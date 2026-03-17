package server

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func GetServerName() string {
	name := os.Getenv("SERVER_NAME")
	if name == "" {
		return "Localspeedtest"
	}
	return name
}

var allowedSizes = map[string]int64{
	"512KB": 512 * 1024,
	"1MB":   1 * 1024 * 1024,
	"5MB":   5 * 1024 * 1024,
	"10MB":  10 * 1024 * 1024,
	"20MB":  20 * 1024 * 1024,
	"50MB":  50 * 1024 * 1024,
	"100MB": 100 * 1024 * 1024,
	"200MB": 200 * 1024 * 1024,
	"500MB": 500 * 1024 * 1024,
	"1GB":   1 * 1024 * 1024 * 1024,
}

func ValidateSize(sizeStr string) (int64, error) {
	sizeStr = strings.ToUpper(sizeStr)
	if val, ok := allowedSizes[sizeStr]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("invalid size")
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	if sizeStr == "" {
		sizeStr = "10MB" // default according to specs
	}

	sizeBytes, err := ValidateSize(sizeStr)
	if err != nil {
		http.Error(w, "Bad Request: Invalid sample size", http.StatusBadRequest)
		return
	}

	ip := GetIP(r)
	LogAction(ip, "START", "DOWNLOAD", sizeStr)
	defer LogAction(ip, "FINISH", "DOWNLOAD", sizeStr)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(sizeBytes, 10))
	w.Header().Set("X-Speedtest-Name", GetServerName())
	w.WriteHeader(http.StatusOK)

	// Write random data efficiently
	buf := make([]byte, 32*1024)
	rand.Read(buf)
	for written := int64(0); written < sizeBytes; {
		toWrite := int64(len(buf))
		if sizeBytes-written < toWrite {
			toWrite = sizeBytes - written
		}
		n, err := w.Write(buf[:toWrite])
		if err != nil {
			break
		}
		written += int64(n)
	}
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	if sizeStr == "" {
		sizeStr = r.Header.Get("X-File-Size")
	}

	_, err := ValidateSize(sizeStr)
	if err != nil {
		http.Error(w, "Bad Request: Invalid sample size", http.StatusBadRequest)
		return
	}

	ip := GetIP(r)
	LogAction(ip, "START", "UPLOAD", sizeStr)
	defer LogAction(ip, "FINISH", "UPLOAD", sizeStr)

	// Discard incoming data body
	io.Copy(io.Discard, r.Body)
	r.Body.Close()
	w.Header().Set("X-Speedtest-Name", GetServerName())
	w.WriteHeader(http.StatusOK)
}
