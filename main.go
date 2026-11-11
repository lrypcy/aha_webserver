// Package main 是 aha_webserver 的启动入口。
//
// @title           aha_webserver API
// @version         1.0
// @description     Job / Task 任务管理 HTTP 服务，提供注册登录、任务创建、分页查询与状态流转。
// @termsOfService  https://opensource.org/license/bsd-3-clause
// @contact.name    aha_webserver maintainers
// @license.name    BSD 3-Clause License
// @license.url     https://opensource.org/license/bsd-3-clause
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in   header
// @name Authorization
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lrypcy/aha_webserver/internal/app"
	"github.com/lrypcy/aha_webserver/internal/config"
	"github.com/lrypcy/aha_webserver/internal/database"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
)

// 由 -ldflags "-X main.version=... -X main.commit=..." 注入。
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := os.Getenv("AHA_CONFIG")
	if cfgPath == "" {
		cfgPath = "configs/config.yml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	setupLogger(cfg.Log.Level)

	if cfg.JWT.EphemeralSecret {
		slog.Warn("using an ephemeral JWT secret; tokens are invalidated on restart. Set AHA_JWT_SECRET for stable tokens.")
	}
	slog.Info("starting aha_webserver",
		"version", version, "commit", commit, "build_time", buildTime,
		"env", cfg.Server.Env, "addr", cfg.Addr(), "database", cfg.Database.Type)

	gdb, err := database.Init(cfg.Database)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(); err != nil {
			slog.Warn("close database", "error", err)
		}
	}()

	tokens, err := jwt.New(cfg.JWT.Secret, cfg.JWT.Expiration, cfg.JWT.Issuer)
	if err != nil {
		return err
	}

	application := app.New(app.Deps{
		Config:    cfg,
		DB:        gdb,
		Tokens:    tokens,
		Version:   version,
		StartedAt: time.Now(),
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownDone := make(chan error, 1)
	go func() {
		<-ctx.Done()
		slog.Info("shutdown signal received", "timeout", cfg.Server.ShutdownTimeout)
		shutdownDone <- application.ShutdownWithTimeout(cfg.Server.ShutdownTimeout)
	}()

	listenErr := application.Listen(cfg.Addr())
	if listenErr != nil && ctx.Err() == nil {
		return fmt.Errorf("listen %s: %w", cfg.Addr(), listenErr)
	}

	select {
	case err := <-shutdownDone:
		if err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		slog.Info("server stopped gracefully")
	case <-time.After(cfg.Server.ShutdownTimeout + time.Second):
		return errors.New("graceful shutdown timed out")
	}
	return nil
}

func setupLogger(level string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}
