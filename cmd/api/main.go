package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	_ "github.com/horaciobranciforte/curiosity-chat-api/docs"
	postgresrepo "github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/repository/postgres"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/handler"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/router"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/auth"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/config"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/database"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/followclient"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/conversation"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
)

//	@title			Curiosity Chat API
//	@version		1.0
//	@description	Real-time 1-on-1 chat microservice for the Curiosity platform. Users who mutually follow each other can exchange messages and share POIs via WebSockets.

//	@host		localhost:8081
//	@BasePath	/api/v1
//	@schemes	http

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and the JWT token. Example: "Bearer eyJhbGci..."

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	logger, err := buildLogger(cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		panic("failed to build logger: " + err.Error())
	}
	defer logger.Sync() //nolint:errcheck

	// Database
	db, err := database.NewPostgresDB(cfg.DSN)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Repositories
	convRepo := postgresrepo.NewConversationRepository(db)
	msgRepo := postgresrepo.NewMessageRepository(db)

	// Follow checker
	var followChecker conversation.FollowChecker
	if cfg.CuriosityAPIURL == "" {
		followChecker = followclient.NoopFollowChecker{}
	} else {
		followChecker = followclient.NewClient(cfg.CuriosityAPIURL)
	}

	// Use cases — conversation
	createConversationUC := conversation.NewCreateConversation(convRepo, followChecker)
	getConversationUC := conversation.NewGetConversation(convRepo)
	listConversationsUC := conversation.NewListConversations(convRepo)

	// Use cases — message
	sendMessageUC := message.NewSendMessage(msgRepo, convRepo)
	getMessagesUC := message.NewGetMessages(msgRepo, convRepo)

	// JWT service
	jwtService := auth.NewJWTService(cfg.JWTSecret)

	// WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Handlers
	healthHandler := handler.NewHealthHandler()
	tokenHandler := handler.NewTokenHandler(jwtService)
	conversationHandler := handler.NewConversationHandler(createConversationUC, getConversationUC, listConversationsUC)
	messageHandler := handler.NewMessageHandler(sendMessageUC, getMessagesUC)
	wsHandler := handler.NewWSHandler(hub, sendMessageUC, convRepo, jwtService, logger)

	// Router
	r := router.NewRouter(healthHandler, tokenHandler, conversationHandler, messageHandler, wsHandler, jwtService)

	// HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", zap.String("port", cfg.ServerPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}

func buildLogger(format, level string) (*zap.Logger, error) {
	var cfg zap.Config
	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return cfg.Build()
}
