package manager

import (
	"context"
	"sync"

	"github.com/logn-xu/gitops-nginx/pkg/log"
)

// Service represents a background service that can be started and stopped.
type Service interface {
	// Start starts the service. It should block until the service stops or the context is canceled.
	Start(ctx context.Context) error
}

// Manager manages the lifecycle of multiple services.
type Manager struct {
	services []Service
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewManager creates a new service manager.
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Add adds a service to the manager.
func (m *Manager) Add(s Service) {
	m.services = append(m.services, s)
}

// Start starts all registered services.
func (m *Manager) Start() {
	for _, s := range m.services {
		m.wg.Add(1)
		go func(s Service) {
			defer m.wg.Done()
			if err := s.Start(m.ctx); err != nil && err != context.Canceled {
				log.Logger.WithError(err).Error("Service stopped with error")
			}
		}(s)
	}
}

// Stop stops all services.
func (m *Manager) Stop() {
	m.cancel()
	m.wg.Wait()
}
