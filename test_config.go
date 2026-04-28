package main

import (
	"fmt"
	"github.com/cpic/record_v2/internal/config"
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
