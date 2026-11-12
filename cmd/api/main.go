package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	_ "github.com/horaciobranciforte/curiosity-chat-api/docs"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/app"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/config"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/database"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
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

	db, err := database.NewPostgresDB(cfg.DSN)
	if err != nil {
		logger.Log.Fatal("failed to connect to database", zap.Error(err))
	}

	application := app.Build(cfg, db)
	defer application.Close()

	// The signal handling is all that is left here: everything the service is made of lives
	// in app.Build, where a test can reach it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		logger.Log.Fatal("server failed", zap.Error(err))
	}
}
