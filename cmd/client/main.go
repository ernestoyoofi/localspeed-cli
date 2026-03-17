package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ernestoyoofi/localspeed-cli/internal/client"
)

func main() {
	fmt.Println("")
	fmt.Println("     Localspeedtest CLI!")
	fmt.Println("")
	unsecure := flag.Bool("unsecure", false, "Skip TLS verification")
	savePath := flag.String("save", "", "Path to save CSV results")
	sampleSize := flag.String("sample", "10MB", "Sample size for testing")
	flag.Usage = func() {
		fmt.Printf("Usage: %s [TARGET_URL] [FLAGS]\n", os.Args[0])
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Error: Target URL is required.")
		flag.Usage()
		os.Exit(1)
	}

	targetURL := args[0]

	client.RunBenchmark(targetURL, *sampleSize, *unsecure, *savePath)
}
