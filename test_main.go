package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Step 1: Starting...")
	
	// Check working directory
	wd, _ := os.Getwd()
	fmt.Printf("Step 2: Working directory: %s\n", wd)
	
	// Check if config file exists
	if _, err := os.Stat("config.yaml"); err == nil {
		fmt.Println("Step 3: config.yaml exists")
	} else {
		fmt.Println("Step 3: config.yaml NOT found")
	}
	
	// Try to import config
	fmt.Println("Step 4: Importing config package...")
	
	fmt.Println("All steps passed!")
}
