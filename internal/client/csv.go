package client

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
)

type ResultRow struct {
	Type       string
	SpeedKBps  float64
	Progress   float64
	JitterMs   float64
	PacketLoss float64
}

type CSVWriter struct {
	file   *os.File
	writer *csv.Writer
	mu     sync.Mutex
}

func NewCSVWriter(path string) (*CSVWriter, error) {
	if path == "" {
		return nil, nil // return nil safely if no path is provided
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	writer := csv.NewWriter(file)
	// Write header
	err = writer.Write([]string{"type", "size_kbps", "progress_percent", "jitter_ms", "packet_loss_percent"})
	if err != nil {
		return nil, err
	}
	writer.Flush()

	return &CSVWriter{
		file:   file,
		writer: writer,
	}, nil
}

func (c *CSVWriter) WriteRow(row ResultRow) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.writer.Write([]string{
		row.Type,
		fmt.Sprintf("%.2f", row.SpeedKBps),
		fmt.Sprintf("%.2f", row.Progress),
		fmt.Sprintf("%.2f", row.JitterMs),
		fmt.Sprintf("%.2f", row.PacketLoss),
	})
	c.writer.Flush()
}

func (c *CSVWriter) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writer.Flush()
	c.file.Close()
}
