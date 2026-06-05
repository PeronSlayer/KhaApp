package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/khaapp/khaapp-daemon/dbus"
	"github.com/khaapp/khaapp-daemon/protocol"
	"github.com/khaapp/khaapp-daemon/storage"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

const Version = "0.1.0-beta.1"

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer func() {
		_ = logger.Sync()
	}()
	logger.Info("KhaApp daemon", zap.String("version", Version))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	storePath, err := resolveStorePath()
	if err != nil {
		logger.Fatal("failed to resolve store path", zap.Error(err))
	}

	container, err := storage.NewStore(storePath)
	if err != nil {
		logger.Fatal("failed to initialize store", zap.Error(err))
	}

	messageStorePath := filepath.Join(filepath.Dir(storePath), "messages.db")
	messageStore, err := storage.NewMessageStore(messageStorePath)
	if err != nil {
		logger.Fatal("failed to initialize message store", zap.Error(err))
	}

	waClient, err := protocol.NewWAClient(container, messageStore, logger)
	if err != nil {
		logger.Fatal("failed to initialize WhatsApp client", zap.Error(err))
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		logger.Fatal("failed to connect to session bus", zap.Error(err))
	}
	defer conn.Close()

	service := dbus.NewMessengerService(conn, waClient, messageStore, logger)
	if err := service.Export(); err != nil {
		logger.Fatal("failed to export D-Bus service", zap.Error(err))
	}

	go service.RunEventLoop(ctx)

	if waClient.HasSession() {
		logger.Info("existing device found, connecting automatically")
		if err := service.AutoConnect(); err != nil {
			logger.Warn("auto-connect failed", zap.Error(err))
		}
	} else {
		logger.Info("no existing session found, waiting for login request")
	}

	<-ctx.Done()

	logger.Info("shutting down daemon")
	service.Shutdown()
	if err := messageStore.Close(); err != nil {
		logger.Warn("failed to close message store", zap.Error(err))
	}
	if err := container.Close(); err != nil {
		logger.Warn("failed to close store container", zap.Error(err))
	}
}

func resolveStorePath() (string, error) {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "khaapp", "session.db"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if homeDir == "" {
		return "", errors.New("user home directory is empty")
	}

	return filepath.Join(homeDir, ".local", "share", "khaapp", "session.db"), nil
}
