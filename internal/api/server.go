package api

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/internal/etcd"
	"github.com/logn-xu/gitops-nginx/internal/ssh"
	"github.com/logn-xu/gitops-nginx/pkg/log"
)

type Server struct {
	cfg        *config.Config
	etcdClient *etcd.Client
	router     *gin.Engine
	sshPools   map[string]*ssh.SFTPPool
	poolsMu    sync.Mutex
}

func NewServer(cfg *config.Config, etcdClient *etcd.Client, dist embed.FS) *Server {
	s := &Server{
		cfg:        cfg,
		etcdClient: etcdClient,
		router:     gin.New(),
		sshPools:   make(map[string]*ssh.SFTPPool),
	}
	s.setupRoutes()
	s.setupStaticRoutes(dist)
	return s
}

// NewServerWithoutUI creates an API server without embedded UI
func NewServerWithoutUI(cfg *config.Config, etcdClient *etcd.Client) *Server {
	s := &Server{
		cfg:        cfg,
		etcdClient: etcdClient,
		router:     gin.New(),
		sshPools:   make(map[string]*ssh.SFTPPool),
	}
	s.setupRoutes()
	return s
}

func (s *Server) getPool(srvCfg *config.ServerConfig) (*ssh.SFTPPool, error) {
	s.poolsMu.Lock()
	defer s.poolsMu.Unlock()

	key := fmt.Sprintf("%s:%d", srvCfg.Host, srvCfg.Port)
	if pool, ok := s.sshPools[key]; ok {
		return pool, nil
	}

	pool, err := ssh.NewSFTPPool(srvCfg, 5) // Default max capacity 5
	if err != nil {
		return nil, err
	}
	s.sshPools[key] = pool
	return pool, nil
}

func (s *Server) setupRoutes() {
	// Custom logger middleware
	s.router.Use(log.GinMiddleware())
	s.router.Use(gin.Recovery())

	// CORS middleware
	s.router.Use(func(c *gin.Context) {
		allowOrigins := s.cfg.API.AllowOrigins
		origin := c.Request.Header.Get("Origin")

		allow := false
		if len(allowOrigins) == 0 {
			allow = true
		} else {
			for _, o := range allowOrigins {
				if o == "*" || o == origin {
					allow = true
					break
				}
			}
		}

		if allow {
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/groups", s.handleGetGroups)
		v1.GET("/tree", s.handleGetTree)
		v1.GET("/triple-diff", s.handleGetTripleDiff)
		v1.POST("/check", s.handleCheckConfig)
		v1.POST("/update/prepare", s.handleUpdatePrepare)
		v1.POST("/update/apply", s.handleUpdateApply)
		v1.GET("/git/status", s.handleGetGitStatus)
	}

}

func (s *Server) setupStaticRoutes(dist embed.FS) {
	// Serve embedded static files if enabled
	// Todo: remove this when we remove the embedded static files
	if s.cfg.API.EnableEmbeddedServer {
		// Get sub-filesystem from embed.FS (strip "dist" prefix)
		subFS, err := fs.Sub(dist, "dist")
		if err != nil {
			log.Logger.Errorf("Failed to get sub filesystem: %v", err)
			return
		}

		// Read index.html content once at startup
		indexHTML, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			log.Logger.Errorf("Failed to read index.html: %v", err)
			return
		}

		// Serve static files from embedded filesystem
		s.router.StaticFS("/assets", http.FS(mustSub(subFS, "assets")))

		// Serve index.html for root
		s.router.GET("/", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
		})

		// SPA fallback: serve index.html for unmatched routes (excluding /api)
		s.router.NoRoute(func(c *gin.Context) {
			// Don't serve index.html for API routes
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
		})

		log.Logger.Info("Embedded static server enabled, serving from embed.FS")
	}
}

// mustSub returns a sub-filesystem or panics on error
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.cfg.API.Listen,
		Handler: s.router,
	}

	go func() {
		<-ctx.Done()
		log.Logger.Info("shutting down API server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Logger.Infof("API server starting on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
