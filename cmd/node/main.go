package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/3fs-csi/3fs-csi/pkg/config"
	"github.com/3fs-csi/3fs-csi/pkg/driver"
	"github.com/3fs-csi/3fs-csi/pkg/mount"
)

func main() {
	var logLevel string
	flag.StringVar(&logLevel, "log-level", "", "log level (debug|info|warn|error)")
	flag.Parse()

	cfg, err := config.FromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse config: %v\n", err)
		os.Exit(1)
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}

	switch cfg.LogLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	logrus.WithFields(logrus.Fields{
		"driverName":       cfg.DriverName,
		"clusterId":        cfg.ClusterID,
		"globalMountBase":  cfg.GlobalMountBase,
		"socket":           cfg.PluginSocketPath(),
		"hf3fsBinaryPath":  cfg.HF3FSBinaryPath,
		"mgmtdAddresses":   cfg.MgmtdAddresses,
		"logLevel":         cfg.LogLevel,
	}).Info("starting 3FS CSI Node plugin")

	go func() {
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		addr := ":9808"
		if err := http.ListenAndServe(addr, nil); err != nil {
			logrus.WithError(err).Warn("health server exited")
		}
	}()

	sockPath := cfg.PluginSocketPath()
	if err := os.MkdirAll(filepath.Dir(sockPath), 0755); err != nil {
		logrus.WithError(err).Fatal("failed creating socket directory")
	}
	_ = os.Remove(sockPath)

	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		logrus.WithError(err).Fatal("failed to listen on unix socket")
	}
	if err := os.Chmod(sockPath, 0600); err != nil {
		logrus.WithError(err).Warn("failed to chmod socket")
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	mounter, err := mount.New(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create mount manager")
	}

	identitySrv := driver.NewIdentityServer(cfg)
	nodeSrv := driver.NewNodeServer(cfg, mounter)

	csi.RegisterIdentityServer(grpcServer, identitySrv)
	csi.RegisterNodeServer(grpcServer, nodeSrv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logrus.Infof("serving CSI on %s", sockPath)
		if err := grpcServer.Serve(lis); err != nil {
			logrus.WithError(err).Fatal("gRPC server terminated")
		}
	}()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sigCh:
		logrus.Infof("received signal: %s, shutting down", s)
		shutdown(grpcServer, ctx)
	case <-ctx.Done():
		shutdown(grpcServer, ctx)
	}
}

func shutdown(s *grpc.Server, ctx context.Context) {
	ch := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(ch)
	}()
	select {
	case <-ch:
		logrus.Info("gRPC server stopped")
	case <-time.After(5 * time.Second):
		logrus.Warn("forcing gRPC stop")
		s.Stop()
	}
}
