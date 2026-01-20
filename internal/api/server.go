package api

import (
	"context"
	"fmt"
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

func NewServer(cfg *config.Config, etcdClient *etcd.Client) *Server {
	s := &Server{
		cfg:        cfg,
		etcdClient: etcdClient,
		router:     gin.Default(),
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

	// Serve UI static files if needed or proxy
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
