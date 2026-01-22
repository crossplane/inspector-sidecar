/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License. You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.
*/

// Package main implements a reference Pipeline Inspector sidecar that logs
// function pipeline execution data to stdout.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	pipelinev1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/pipelineinspector/proto/v1alpha1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/inspector-sidecar/server"
)

// CLI arguments.
type CLI struct {
	Debug           bool          `help:"Emit debug logs in addition to info logs." short:"d"`
	SocketPath      string        `default:"/var/run/pipeline-inspector/socket"     env:"PIPELINE_INSPECTOR_SOCKET" help:"Unix socket path to listen on."`
	Format          string        `default:"json"                                   enum:"json,text"                help:"Output format (json or text)."`
	MaxRecvMsgSize  int           `default:"4194304"                                env:"MAX_RECV_MSG_SIZE"         help:"Maximum gRPC receive message size in bytes (default 4MB)."`
	ShutdownTimeout time.Duration `default:"5s"                                     env:"SHUTDOWN_TIMEOUT"          help:"Graceful shutdown timeout."`
}

func main() {
	var cli CLI
	kong.Parse(&cli)

	if err := run(cli); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(cli CLI) error {
	// Create logger.
	log, err := newLogger(cli.Debug)
	if err != nil {
		return fmt.Errorf("cannot create logger: %w", err)
	}

	// Ensure socket directory exists.
	socketDir := filepath.Dir(cli.SocketPath)
	if err := os.MkdirAll(socketDir, 0o750); err != nil {
		return fmt.Errorf("cannot create socket directory: %w", err)
	}

	// Remove existing socket file if it exists.
	if err := os.Remove(cli.SocketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove existing socket: %w", err)
	}

	// Listen on Unix socket.
	lc := net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "unix", cli.SocketPath)
	if err != nil {
		return fmt.Errorf("cannot listen on socket: %w", err)
	}
	defer func() { _ = listener.Close() }()

	log.Info("Pipeline Inspector listening", "socket", cli.SocketPath, "format", cli.Format)

	// Create gRPC server.
	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(cli.MaxRecvMsgSize))
	inspector := server.NewInspector(cli.Format, server.WithLogger(log))
	pipelinev1alpha1.RegisterPipelineInspectorServiceServer(grpcServer, inspector)

	// Handle shutdown signals.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		log.Info("Shutting down")

		// Create a timeout context for graceful shutdown.
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cli.ShutdownTimeout)
		defer shutdownCancel()

		// Try graceful shutdown first.
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-shutdownCtx.Done():
			log.Info("Graceful shutdown timed out, forcing stop")
			grpcServer.Stop()
		case <-stopped:
			// Graceful shutdown completed.
		}
	}()

	// Serve requests.
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// newLogger creates a new logger based on the debug flag.
func newLogger(debug bool) (logging.Logger, error) {
	var zl *zap.Logger
	var err error

	if debug {
		zl, err = zap.NewDevelopment()
	} else {
		zl, err = zap.NewProduction()
	}
	if err != nil {
		return nil, err
	}

	return logging.NewLogrLogger(zapr.NewLogger(zl)), nil
}
