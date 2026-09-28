package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"restapi-test/internal/handler"
	"restapi-test/internal/middleware"
	"restapi-test/internal/response"

	_ "restapi-test/docs"
)

type Deps struct {
	Redis           *redis.Client
	RateLimitPrefix string
	Products        *handler.ProductHandler
	Auth            *handler.AuthHandler
	AuthMiddleware  gin.HandlerFunc
}

func New(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestLogger(), middleware.CORS())
	r.HandleMethodNotAllowed = true

	prefix := deps.RateLimitPrefix
	if prefix == "" {
		prefix = "rl"
	}

	authLimit := middleware.RateLimitRule{Limit: 3, Window: 60 * time.Second}
	writeLimit := middleware.RateLimitRule{Limit: 1, Window: 5 * time.Second}

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "endpoint tidak ditemukan", map[string]any{
			"path": c.Request.URL.Path,
		})
	})
	r.NoMethod(func(c *gin.Context) {
		response.Fail(c, http.StatusMethodNotAllowed, response.CodeNotFound, "metode tidak diizinkan", map[string]any{
			"method": c.Request.Method,
		})
	})

	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", middleware.RateLimit(deps.Redis, prefix, "auth", authLimit), deps.Auth.Register)
			authGroup.POST("/login", middleware.RateLimit(deps.Redis, prefix, "auth", authLimit), deps.Auth.Login)
			authGroup.POST("/refresh", middleware.RateLimit(deps.Redis, prefix, "auth", authLimit), deps.Auth.Refresh)
			authGroup.POST("/logout", middleware.RateLimit(deps.Redis, prefix, "auth", authLimit), deps.Auth.Logout)
		}

		productGroup := api.Group("/products")
		{
			productGroup.GET("", deps.Products.List)
			productGroup.GET("/:id", deps.Products.GetByID)

			authorized := productGroup.Group("", deps.AuthMiddleware)
			{
				authorized.POST("", middleware.RateLimit(deps.Redis, prefix, "product-write", writeLimit), deps.Products.Create)
				authorized.PUT("/:id", middleware.RateLimit(deps.Redis, prefix, "product-write", writeLimit), deps.Products.Update)
				authorized.DELETE("/:id", middleware.RateLimit(deps.Redis, prefix, "product-write", writeLimit), deps.Products.Delete)
			}
		}
	}

	return r
}
