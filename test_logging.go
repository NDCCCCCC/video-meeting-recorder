package main

import (
	"fmt"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/logging"
)

func main() {
	fmt.Println("Loading config...")
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Println("Initializing logger...")
	logger, err := logging.New(cfg.Logging, cfg.Server.Environment)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	logger.Info("Test log message")
	fmt.Println("Logging works!")
}
