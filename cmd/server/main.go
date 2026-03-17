package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"fmt"

	"github.com/ernestoyoofi/localspeed-cli/internal/server"
)

func main() {
	fmt.Println("")
	fmt.Println("     Localspeedtest Server!")
	fmt.Println("")
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "7520"
	}
	enableTLS := os.Getenv("SERVER_ENABLE_TLS") == "true"
	certFile := os.Getenv("SERVER_TLS_CERT")
	keyFile := os.Getenv("SERVER_TLS_KEY")

	http.HandleFunc("/test/download", server.DownloadHandler)
	http.HandleFunc("/test/upload", server.UploadHandler)
	http.HandleFunc("/test/ws", server.WsHandler)

	log.Printf("Starting Server on :%s (TLS: %v)\n", port, enableTLS)

	if enableTLS && certFile != "" && keyFile != "" {
		serverConfig := &http.Server{
			Addr: ":" + port,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		err := serverConfig.ListenAndServeTLS(certFile, keyFile)
		if err != nil {
			log.Fatalf("TLS server failed: %v", err)
		}
	} else {
		err := http.ListenAndServe(":"+port, nil)
		if err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
