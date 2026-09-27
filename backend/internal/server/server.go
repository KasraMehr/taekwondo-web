package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"backend/internal/database"
	"backend/internal/platform"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct{}

func NewRouter(pool *pgxpool.Pool) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	// Proxy forwarding headers are not trusted unless deployment adds explicit proxies.
	_ = router.SetTrustedProxies(nil)
	if origins := strings.TrimSpace(os.Getenv("CORS_ORIGINS")); origins != "" {
		router.Use(cors.New(cors.Config{AllowOrigins: strings.Split(origins, ","), AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Authorization", "Content-Type", "If-Match"}, MaxAge: 12 * time.Hour}))
	}
	router.GET("/", (&Server{}).HelloWorldHandler)
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			c.JSON(503, gin.H{"status": "down"})
			return
		}
		c.JSON(200, gin.H{"status": "up"})
	})
	(&platform.API{DB: pool}).Routes(router)
	return router
}
func NewServer() *http.Server {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, database.ConnectionString())
	if err != nil {
		log.Fatal("invalid database configuration")
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		log.Fatal("database unavailable")
	}
	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err = database.Migrate(ctx, pool); err != nil {
			pool.Close()
			log.Fatal(err)
		}
	}
	port := 8080
	if n, err := strconv.Atoi(os.Getenv("PORT")); err == nil && n > 0 && n <= 65535 {
		port = n
	}
	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: NewRouter(pool), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: time.Minute}
	server.RegisterOnShutdown(pool.Close)
	return server
}
