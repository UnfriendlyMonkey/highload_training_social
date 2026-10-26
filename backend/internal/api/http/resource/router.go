package resource

import (
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/handler"
	"github.com/UnfriendlyMonkey/hsn/internal/api/http/middleware"
	"github.com/UnfriendlyMonkey/hsn/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(userSvc *service.UserService, authSvc *service.AuthService, loadTestSvc *service.LoadTestService, socialSvc *service.SocialService, tokens middleware.TokenLookup) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(chimw.Logger)

	userHandler := handler.NewUserHandler(userSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	loadTestHandler := handler.NewLoadTestHandler(loadTestSvc)
	socialHandler := handler.NewSocialHandler(socialSvc)

	r.Post("/user/register", userHandler.Register)
	r.Get("/user/get/{id}", userHandler.Get)
	r.Get("/user/search", userHandler.Search)
	r.Post("/login", authHandler.Login)
	r.Get("/post/get/{id}", socialHandler.GetPost)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(tokens))
		r.Put("/friend/set/{user_id}", socialHandler.SetFriend)
		r.Put("/friend/delete/{user_id}", socialHandler.DeleteFriend)
		r.Post("/post/create", socialHandler.CreatePost)
		r.Put("/post/update", socialHandler.UpdatePost)
		r.Put("/post/delete/{id}", socialHandler.DeletePost)
		r.Get("/post/feed", socialHandler.GetFeed)
	})
	r.Post("/load-test/event", loadTestHandler.RecordEvent)
	r.Get("/load-test/count", loadTestHandler.Count)

	return r
}
