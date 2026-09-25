package main

import (
	"fmt"
	"os"

	"github.com/officel/uchi/internal/app"
	"github.com/officel/uchi/internal/color"
	"github.com/officel/uchi/internal/config"
)

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		c := color.New(color.ModeAuto, os.Stderr)
		fmt.Fprintf(os.Stderr, "%s\n", c.Red(fmt.Sprintf("Error loading configuration: %v", err)))
		os.Exit(1)
	}
	if err := app.Run(cfg, os.Stdout); err != nil {
		c := color.New(cfg.Color, os.Stderr)
		fmt.Fprintf(os.Stderr, "%s\n", c.Red(fmt.Sprintf("Error executing command: %v", err)))
		os.Exit(1)
	}
}
