package main

import (
	"context"
	"github.com/serhiq/tiny-phone-linker/internal/app"
	"github.com/serhiq/tiny-phone-linker/internal/config"
	"github.com/serhiq/tiny-phone-linker/internal/logger"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	defer func() {
		logger.Sync()
	}()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(err)
	}

	var server = app.New(cfg)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	go server.Start()

	select {
	case <-stopChan:
		logger.Info("Exit signal received.")
		logger.Debug("Goroutines:", zap.Any("num goroutines", runtime.NumGoroutine()))
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	case err := <-server.ErrChan:
		logger.Fatal(err.Error())
	}
}
