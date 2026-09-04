package main

import (
	"fmt"
	"os"
)

func main() {
	cfg, err := LoadConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if err := Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
