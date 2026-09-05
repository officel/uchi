package main

import (
	"fmt"
	"os"

	"github.com/officel/uchi/internal/app"
	"github.com/officel/uchi/internal/config"
)

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(cfg, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
