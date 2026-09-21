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
	"oncode/local-agent/internal/idebridge"
	"oncode/local-agent/internal/idesession"
	"oncode/local-agent/internal/transport"
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

	hubOpts := idesession.Options{WorkspaceID: "ws-local"}
	if *workspacePath != "" {
		ws, err := a.Workspaces.Register("", *workspacePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "register workspace: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "registered workspace %s root=%s\n", ws.ID, ws.RootPath)
		hubOpts.WorkspaceID = ws.ID
		hubOpts.Workspaces = a.Workspaces
		hubOpts.Proposals = a.Proposals
		hubOpts.MaxFileBytes = cfg.MaxFileBytes
	}

	fmt.Fprintf(os.Stderr, "oncode-agent %s id=%s protocol=oncode-tool/1.0 tools=%v\n",
		cfg.AgentVersion, a.Cfg.AgentID, a.Tools.Names())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var stream *transport.Stream
	if cfg.ServerGRPCAddr != "" {
		stream = transport.NewStream(cfg.ServerGRPCAddr, a.Cfg.AgentID)
		if err := stream.Connect(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "server grpc: %v (chat unavailable until reconnect)\n", err)
			_ = stream.Close()
			stream = nil
		} else {
			if err := stream.Register(ctx, a.Cfg.AgentID); err != nil {
				fmt.Fprintf(os.Stderr, "server register: %v\n", err)
			}
			go func() {
				// Serve는 Connect를 다시 해서 스트림을 두 개 연다. 이미 붙은 연결에서 도구 REQUEST만 처리한다.
				_ = transport.Loop(ctx, stream, a.HandleEnvelope)
			}()
			fmt.Fprintf(os.Stderr, "server grpc connected %s\n", cfg.ServerGRPCAddr)
		}
	}

	bridge, err := idebridge.New(idebridge.Options{
		Bind:     cfg.IdeIPCBind,
		Port:     cfg.IdeIPCPort,
		StateDir: cfg.StateDir,
		AgentID:  a.Cfg.AgentID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ide ipc: %v\n", err)
		os.Exit(1)
	}
	hubOpts.Push = bridge.Broadcast
	if stream != nil {
		hubOpts.Stream = stream
	}
	hub := idesession.NewWithOptions(hubOpts)
	bridge.SetHandler(hub.Handle)
	if err := bridge.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ide ipc listen: %v\n", err)
		os.Exit(1)
	}
	defer bridge.Close()
	go hub.Listen(ctx)
	if ep, err := bridge.Endpoint(); err == nil {
		fmt.Fprintf(os.Stderr, "ide ipc listening on 127.0.0.1:%d%s (token in %s)\n",
			ep.Port, ep.Path, cfg.StateDir)
	}

	<-ctx.Done()
	if stream != nil {
		_ = stream.Close()
	}
}
