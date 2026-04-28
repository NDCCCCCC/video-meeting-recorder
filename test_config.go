package main

import (
	"fmt"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
)

func main() {
	fmt.Println("Loading config...")
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Config loaded: %+v\n", cfg.Server)
}
