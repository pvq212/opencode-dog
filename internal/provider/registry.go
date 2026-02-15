package provider

import (
	"log/slog"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[ProviderType]Provider
	logger    *slog.Logger
}

func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		providers: make(map[ProviderType]Provider),
		logger:    logger,
	}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Type()] = p
	r.logger.Info("provider registered", "type", p.Type())
}

func (r *Registry) Get(t ProviderType) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[t]
	return p, ok
}

func (r *Registry) All() map[ProviderType]Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[ProviderType]Provider, len(r.providers))
	for k, v := range r.providers {
		out[k] = v
	}
	return out
}
