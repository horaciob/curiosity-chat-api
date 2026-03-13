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
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/handler"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/router"
	postgresrepo "github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/repository/postgres"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/authclient"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/config"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/database"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/followclient"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
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

	if err := logger.Init(logger.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	}); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Database
	db, err := database.NewPostgresDB(cfg.DSN)
	if err != nil {
		logger.Log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Repositories
	convRepo := postgresrepo.NewConversationRepository(db)
	msgRepo := postgresrepo.NewMessageRepository(db)

	// Follow checker
	var followChecker conversation.FollowChecker
	if cfg.UserAPIURL == "" {
		followChecker = followclient.NoopFollowChecker{}
	} else {
		followChecker = followclient.NewClient(cfg.UserAPIURL, cfg.InternalAPIKey)
	}

	// Use cases — conversation
	createConversationUC := conversation.NewCreateConversation(convRepo)
	getConversationUC := conversation.NewGetConversation(convRepo)
	listConversationsUC := conversation.NewListConversations(convRepo)

	// Use cases — message
	sendMessageUC := message.NewSendMessage(msgRepo, convRepo, followChecker)
	getMessagesUC := message.NewGetMessages(msgRepo, convRepo)

	// Auth client — delegates token validation to curiosity-user-api
	authClient := authclient.NewClient(cfg.UserAPIURL, cfg.InternalAPIKey)

	// WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Handlers
	healthHandler := handler.NewHealthHandler()
	conversationHandler := handler.NewConversationHandler(createConversationUC, getConversationUC, listConversationsUC)
	messageHandler := handler.NewMessageHandler(sendMessageUC, getMessagesUC)
	wsHandler := handler.NewWSHandler(hub, sendMessageUC, convRepo, msgRepo, authClient, logger.Log)

	// Router
	r := router.NewRouter(healthHandler, conversationHandler, messageHandler, wsHandler, authClient, cfg.InternalAPIKey, cfg.AllowedOrigins)

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
