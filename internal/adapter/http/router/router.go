package router

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/handler"
	custommiddleware "github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/middleware"
)

func NewRouter(
	healthHandler *handler.HealthHandler,
	conversationHandler *handler.ConversationHandler,
	messageHandler *handler.MessageHandler,
	wsHandler *handler.WSHandler,
	tokenValidator custommiddleware.TokenValidator,
	internalAPIKey string,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Internal-Key"},
	}))

	// Swagger UI
	r.Get("/swagger/doc.json", swaggerDocHandler)
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	r.Get("/swagger/", swaggerUIHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthHandler.Health)

		// WebSocket — auth via first frame (no middleware needed)
		r.Get("/ws", wsHandler.ServeWS)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(custommiddleware.Authenticate(tokenValidator))

			r.Route("/conversations", func(r chi.Router) {
				r.Post("/", conversationHandler.Create)
				r.Get("/", conversationHandler.List)
				r.Get("/{id}", conversationHandler.Get)
				r.Get("/{id}/messages", messageHandler.List)
				r.Post("/{id}/messages", messageHandler.Send)
			})
		})
	})

	return r
}

func swaggerDocHandler(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		http.Error(w, "swagger.json not found — run make docs", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data) //nolint:errcheck
}

func swaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
  <title>Curiosity Chat API — Swagger UI</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
  SwaggerUIBundle({
    url: "/swagger/doc.json",
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
    layout: "BaseLayout",
    deepLinking: true,
  })
</script>
</body>
</html>`)
}
