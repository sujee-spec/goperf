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

	res, err := engine.Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	report.Print(os.Stdout, cfg, res)
}
