package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ambientlabscomputing/cron_engine/internal/api"
	"github.com/ambientlabscomputing/cron_engine/internal/syscall"
	"github.com/ambientlabscomputing/umc_sdk/lifecycle"
	"github.com/ambientlabscomputing/umc_sdk/logging"
	"github.com/ambientlabscomputing/umc_sdk/middleware"
	"github.com/ambientlabscomputing/umc_sdk/transport"
	"google.golang.org/grpc"
)

func main() {
	logger, err := logging.Setup(logging.Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup logging: %v\n", err)
		os.Exit(1)
	}

	conn, err := dialKernelSyscallServer(5 * time.Second)
	if err != nil {
		logger.Error("failed to connect to kernel syscall server", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	syscallClient := syscall.NewClient(conn, logger)
	logger.Info("kernel syscall client connected")

	apiHandler := api.NewHandler(syscallClient, logger)

	// Wrap with trace middleware
	wrappedHandler := middleware.TraceIDMiddleware(apiHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "10082"
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      wrappedHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	launcher := lifecycle.NewLauncher(logger)
	launcher.Add(&HTTPServerComponent{
		server: httpServer,
		logger: logger,
	})

	runtime := lifecycle.NewRuntime(launcher)
	if err := runtime.Run(); err != nil {
		logger.Error("runtime error", "error", err)
		os.Exit(1)
	}
}

type HTTPServerComponent struct {
	server *http.Server
	logger *slog.Logger
}

func (c *HTTPServerComponent) Name() string {
	return "http-server"
}

func (c *HTTPServerComponent) Start(ctx context.Context) error {
	c.logger.Info("starting http server", "addr", c.server.Addr)
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.logger.Error("http server error", "error", err)
		}
	}()
	return nil
}

func (c *HTTPServerComponent) Stop(ctx context.Context) error {
	return c.server.Shutdown(ctx)
}

func dialKernelSyscallServer(timeout time.Duration) (*grpc.ClientConn, error) {
	socketPath := os.Getenv("KERNEL_SOCKET")
	if socketPath == "" {
		socketPath = "/tmp/ua_kernel.sock"
	}

	conn, err := transport.UDSDialer(socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to dial kernel syscall server at %s: %w", socketPath, err)
	}

	return conn, nil
}
