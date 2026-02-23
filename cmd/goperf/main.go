package main

import (
	"fmt"
	"os"

	"goperf/internal/config"
	"goperf/internal/engine"
	"goperf/internal/report"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	res := engine.Run(cfg)

	if err := report.Print(os.Stdout, cfg, res); err != nil {
		fmt.Fprintf(os.Stderr, "error writing report: %v\n", err)
		os.Exit(1)
	}
}
