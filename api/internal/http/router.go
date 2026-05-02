package httpapi

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"trade-buddy/api/internal/config"
)

type Dependencies struct {
	Config config.Config
	DB     *gorm.DB
}

type Server struct {
	cfg config.Config
	db  *gorm.DB
}

func NewRouter(deps Dependencies) *gin.Engine {
	server := &Server{cfg: deps.Config, db: deps.DB}

	r := gin.Default()
	r.Use(withCommonHeaders())
	r.GET("/health", server.health)

	return r
}

func withCommonHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}
