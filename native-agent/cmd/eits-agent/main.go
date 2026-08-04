package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/eitsio/eits-agent/internal/agent"
	"github.com/eitsio/eits-agent/internal/config"
	elog "github.com/eitsio/eits-agent/internal/logging"
)

const version = "0.3.0"

func usage() {
	fmt.Fprintf(os.Stderr, "EITS Monitoring Agent %s\n\n", version)
	fmt.Fprintln(os.Stderr, "Usage: eits-agent [run|once|diagnostics|version] [-config PATH]")
}

func main() {
	command := "run"
	if len(os.Args) > 1 && os.Args[1][0] != '-' {
		command = os.Args[1]
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	_ = fs.Parse(os.Args[1:])
	if command == "version" {
		fmt.Println(version)
		return
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	logger := log.New(elog.Multi(filepath.Join(cfg.LogDirectory, "eits-agent.log"), cfg.Logging.MaximumSizeMB, cfg.Logging.MaximumFiles), "", log.Ldate|log.Ltime|log.LUTC)
	runtimeAgent, err := agent.New(*configPath, version, logger)
	if err != nil {
		logger.Fatalf("startup error: %v", err)
	}
	switch command {
	case "run":
		stop := make(chan struct{})
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		go func() { <-sig; close(stop) }()
		if err := runtimeAgent.Run(stop); err != nil {
			logger.Fatalf("agent stopped: %v", err)
		}
	case "once":
		if err := runtimeAgent.Once(); err != nil {
			logger.Fatalf("one-shot report failed: %v", err)
		}
	case "diagnostics":
		if err := runtimeAgent.Diagnostics(); err != nil {
			logger.Fatalf("diagnostics failed: %v", err)
		}
	default:
		usage()
		os.Exit(2)
	}
}
