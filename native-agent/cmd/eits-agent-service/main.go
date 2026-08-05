//go:build windows

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/eitsio/eits-agent/internal/agent"
	"github.com/eitsio/eits-agent/internal/config"
	elog "github.com/eitsio/eits-agent/internal/logging"
	"github.com/kardianos/service"
)

var version = "0.5.0-alpha.1"

type program struct {
	configPath string
	stop       chan struct{}
	done       chan struct{}
	logger     *log.Logger
}

func (p *program) Start(service.Service) error {
	p.stop = make(chan struct{})
	p.done = make(chan struct{})
	go p.run()
	return nil
}

func (p *program) run() {
	defer close(p.done)
	cfg, err := config.Load(p.configPath)
	if err != nil {
		p.logger.Printf("configuration error: %v", err)
		return
	}
	runtimeAgent, err := agent.New(p.configPath, version, p.logger)
	if err != nil {
		p.logger.Printf("startup error: %v", err)
		return
	}
	p.logger.Printf("Windows service starting; server=%s", cfg.ServerURL)
	if err := runtimeAgent.Run(p.stop); err != nil {
		p.logger.Printf("agent stopped with error: %v", err)
	}
}

func (p *program) Stop(service.Service) error {
	if p.stop != nil {
		close(p.stop)
	}
	if p.done != nil {
		<-p.done
	}
	return nil
}

func main() {
	configPath := config.DefaultPath()
	fs := flag.NewFlagSet("eits-agent-service", flag.ContinueOnError)
	fs.StringVar(&configPath, "config", configPath, "path to config.json")
	_ = fs.Parse(os.Args[1:])

	cfg := &service.Config{
		Name:        "EITSAgent",
		DisplayName: "EITS Monitoring Agent",
		Description: "Collects and reports system statistics to the EITS Monitor server.",
		Arguments:   []string{"-config", configPath},
		Option: service.KeyValue{
			"StartType": "automatic",
			"OnFailure": "restart",
		},
	}

	logDir := filepath.Join(filepath.Dir(configPath), "logs")
	_ = os.MkdirAll(logDir, 0750)
	logger := log.New(elog.Multi(filepath.Join(logDir, "eits-agent.log"), 10, 5), "", log.Ldate|log.Ltime|log.LUTC)
	prg := &program{configPath: configPath, logger: logger}
	svc, err := service.New(prg, cfg)
	if err != nil {
		logger.Fatal(err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install", "uninstall", "start", "stop", "restart":
			if err := service.Control(svc, os.Args[1]); err != nil {
				logger.Fatalf("service %s failed: %v", os.Args[1], err)
			}
			fmt.Printf("EITSAgent %s completed\n", os.Args[1])
			return
		case "version":
			fmt.Println(version)
			return
		}
	}
	if err := svc.Run(); err != nil {
		logger.Fatal(err)
	}
}
