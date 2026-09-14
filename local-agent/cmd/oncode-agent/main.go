package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"oncode/local-agent/internal/agent"
	"oncode/local-agent/internal/config"
)

func main() {
	workspacePath := flag.String("workspace", "", "optional workspace root to register at startup")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	a, err := agent.New(cfg, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: %v\n", err)
		os.Exit(1)
	}

	if *workspacePath != "" {
		ws, err := a.Workspaces.Register("", *workspacePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "register workspace: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "registered workspace %s root=%s\n", ws.ID, ws.RootPath)
	}

	fmt.Fprintf(os.Stderr, "oncode-agent %s id=%s protocol=oncode-tool/1.0 tools=%v\n",
		cfg.AgentVersion, a.Cfg.AgentID, a.Tools.Names())
	fmt.Fprintf(os.Stderr, "Phase 1 skeleton ready (no live server transport yet). Use go test ./local-agent/...\n")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}
