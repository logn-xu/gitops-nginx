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

// ReloadableService marks a service that can be hot-reloaded.
type ReloadableService interface {
	Service
	Reloadable() bool
}

// Manager manages the lifecycle of multiple services.
type Manager struct {
	services           []Service
	reloadableServices []Service
	wg                 sync.WaitGroup
	reloadWg           sync.WaitGroup
	ctx                context.Context
	cancel             context.CancelFunc
	reloadCtx          context.Context
	reloadCancel       context.CancelFunc
	mu                 sync.Mutex
}

// NewManager creates a new service manager.
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	reloadCtx, reloadCancel := context.WithCancel(ctx)
	return &Manager{
		ctx:          ctx,
		cancel:       cancel,
		reloadCtx:    reloadCtx,
		reloadCancel: reloadCancel,
	}
}

// Add adds a service to the manager.
func (m *Manager) Add(s Service) {
	m.services = append(m.services, s)
}

// AddReloadable adds a reloadable service to the manager.
func (m *Manager) AddReloadable(s Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadableServices = append(m.reloadableServices, s)
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
	m.startReloadableServices()
}

func (m *Manager) startReloadableServices() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.reloadableServices {
		m.reloadWg.Add(1)
		go func(s Service) {
			defer m.reloadWg.Done()
			if err := s.Start(m.reloadCtx); err != nil && err != context.Canceled {
				log.Logger.WithError(err).Error("Reloadable service stopped with error")
			}
		}(s)
	}
}

// Reload stops all reloadable services and starts new ones from the factory.
func (m *Manager) Reload(factory func() []Service) {
	log.Logger.Info("reloading services...")

	// Stop reloadable services
	m.reloadCancel()
	m.reloadWg.Wait()

	// Clear old reloadable services
	m.mu.Lock()
	m.reloadableServices = nil
	m.mu.Unlock()

	// Create new context for reloadable services
	m.reloadCtx, m.reloadCancel = context.WithCancel(m.ctx)

	// Get new services from factory
	newServices := factory()
	for _, s := range newServices {
		m.AddReloadable(s)
	}

	// Start new reloadable services
	m.startReloadableServices()
	log.Logger.Info("services reloaded successfully")
}

// Stop stops all services.
func (m *Manager) Stop() {
	m.cancel()
	m.reloadWg.Wait()
	m.wg.Wait()
}
