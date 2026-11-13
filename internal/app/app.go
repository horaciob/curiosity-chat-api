// Package app assembles the service.
//
// The wiring used to live inside main(), where nothing could reach it: a test cannot call
// main, so every dependency decision — which repository, which client, what the router ends
// up holding — was unreachable from a test and counted as untested code. Moving it here makes
// the assembly an ordinary function that returns an ordinary value.
package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/internalkeys"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/jwks"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/handler"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/router"
	postgresrepo "github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/repository/postgres"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/config"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/jwtauth"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/userclient"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/conversation"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
)

// App is a built service, ready to run.
type App struct {
	Server *http.Server
	Hub    *ws.Hub

	db *sqlx.DB
}

// Build wires everything together from configuration and an open database.
//
// The database is a parameter rather than something Build opens, so a test can hand it a
// sqlmock connection and assert on the result without needing Postgres.
func Build(cfg *config.Config, db *sqlx.DB) *App {
	convRepo := postgresrepo.NewConversationRepository(db)
	msgRepo := postgresrepo.NewMessageRepository(db)

	createConversationUC := conversation.NewCreateConversation(convRepo)
	getConversationUC := conversation.NewGetConversation(convRepo)
	listConversationsUC := conversation.NewListConversations(convRepo)

	sendMessageUC := message.NewSendMessage(msgRepo, convRepo)
	getMessagesUC := message.NewGetMessages(msgRepo, convRepo)

	// Tokens are validated locally — with user-api's public keys (JWKS, EdDSA) and/or the
	// shared secret (HS256) — with no round trip on every request and WebSocket connect.
	// config.Load already refuses a config with neither; an empty one here accepts no token.
	tokenKeys, err := jwks.New(cfg.JWTSecret, cfg.JWTJWKSURL)
	if err != nil {
		tokenKeys = jwks.Secret("")
	}
	if cfg.JWTJWKSURL != "" {
		if err := tokenKeys.Refresh(context.Background()); err != nil {
			logger.Log.Warn("JWKS not reachable yet; will retry on first token", zap.Error(err))
		}
	}
	tokens := jwtauth.NewWithKeys(tokenKeys)

	// Inbound X-Internal-Key values: the shared one plus INTERNAL_API_KEYS (name:key,...).
	internalKeys, err := internalkeys.Parse(cfg.InternalAPIKey, os.Getenv("INTERNAL_API_KEYS"))
	if err != nil {
		panic(err)
	}
	logger.Log.Info("internal callers accepted", zap.Strings("callers", internalKeys.Callers()))

	hub := ws.NewHub()
	// With REDIS_URL, replicas share deliveries and presence (see ws.Cluster); without it
	// this is a single-replica hub, as before.
	if cfg.RedisURL != "" {
		cluster, err := ws.NewRedisCluster(cfg.RedisURL)
		if err != nil {
			panic(err)
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := cluster.Ping(pingCtx); err != nil {
			logger.Log.Warn("Redis not reachable yet; deliveries stay local until it is", zap.Error(err))
		}
		cancel()
		hub.WithCluster(cluster)
		logger.Log.Info("chat hub clustered over Redis")
	}

	healthHandler := handler.NewHealthHandler()
	// user-api resolves names for the conversation list and sends pushes to recipients
	// who are offline.
	userClient := userclient.NewClient(cfg.UserAPIURL, cfg.InternalAPIKey)
	conversationHandler := handler.NewConversationHandler(
		createConversationUC, getConversationUC, listConversationsUC).
		WithSummary(conversation.NewSummarize(msgRepo, userClient))
	messageHandler := handler.NewMessageHandler(
		sendMessageUC, getMessagesUC, hub, convRepo, msgRepo).WithChatPusher(userClient)
	wsHandler := handler.NewWSHandler(
		hub, sendMessageUC, convRepo, msgRepo, tokens, logger.Log, cfg.AllowedOrigins).WithChatPusher(userClient)
	internalHandler := handler.NewInternalHandler(hub)

	r := router.NewRouter(
		healthHandler, conversationHandler, messageHandler, wsHandler, internalHandler,
		tokens, internalKeys, cfg.AllowedOrigins,
	)

	return &App{
		Hub: hub,
		db:  db,
		Server: &http.Server{
			Addr:         ":" + cfg.ServerPort,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Run serves until the context is cancelled, then shuts down gracefully.
//
// Returning an error rather than calling Fatal means a caller — including a test — decides
// what a startup failure means, instead of the process exiting from inside a library.
func (a *App) Run(ctx context.Context) error {
	go a.Hub.Run()
	a.Hub.StartCluster(ctx, func(err error) { logger.Log.Warn("chat cluster", zap.Error(err)) })

	errs := make(chan error, 1)
	go func() {
		logger.Info("server starting", zap.String("addr", a.Server.Addr))
		if err := a.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- fmt.Errorf("server error: %w", err)
			return
		}
		errs <- nil
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}

// Close releases what Build was given.
func (a *App) Close() error {
	if a.db == nil {
		return nil
	}
	return a.db.Close()
}
