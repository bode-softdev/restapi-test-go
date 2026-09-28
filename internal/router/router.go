package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"restapi-test/internal/handler"
	"restapi-test/internal/middleware"
	"restapi-test/internal/response"

	_ "restapi-test/docs"
)

type Deps struct {
	Products       *handler.ProductHandler
	Auth           *handler.AuthHandler
	AuthMiddleware gin.HandlerFunc
}

func New(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestLogger(), middleware.CORS())
	r.HandleMethodNotAllowed = true

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
			authGroup.POST("/register", deps.Auth.Register)
			authGroup.POST("/login", deps.Auth.Login)
			authGroup.POST("/refresh", deps.Auth.Refresh)
			authGroup.POST("/logout", deps.Auth.Logout)
		}

		productGroup := api.Group("/products", deps.AuthMiddleware)
		{
			productGroup.GET("", deps.Products.List)
			productGroup.GET("/:id", deps.Products.GetByID)
			productGroup.POST("", deps.Products.Create)
			productGroup.PUT("/:id", deps.Products.Update)
			productGroup.DELETE("/:id", deps.Products.Delete)
		}
	}

	return r
}
