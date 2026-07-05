package resource

import (
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/handler"
	"github.com/UnfriendlyMonkey/hsn/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(userSvc *service.UserService, authSvc *service.AuthService, loadTestSvc *service.LoadTestService) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(chimw.Logger)

	userHandler := handler.NewUserHandler(userSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	loadTestHandler := handler.NewLoadTestHandler(loadTestSvc)

	r.Post("/user/register", userHandler.Register)
	r.Get("/user/get/{id}", userHandler.Get)
	r.Get("/user/search", userHandler.Search)
	r.Post("/login", authHandler.Login)
	r.Post("/load-test/event", loadTestHandler.RecordEvent)
	r.Get("/load-test/count", loadTestHandler.Count)

	return r
}
