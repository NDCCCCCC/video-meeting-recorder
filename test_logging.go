package main

import (
	"fmt"
	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/logging"
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
