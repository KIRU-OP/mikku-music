package music

import (
	"context"
	"errors"
)

var ErrNotImplemented = errors.New("music provider adapter is not implemented")

// Track is the provider-neutral metadata model used by future source adapters.
type Track struct {
	ID        string
	Title     string
	Artist    string
	Duration  int
	Source    string
	PageURL   string
	StreamURL string
}

// Provider is the contract each source adapter must implement.
type Provider interface {
	Name() string
	Search(ctx context.Context, query string, limit int) ([]Track, error)
	Resolve(ctx context.Context, input string) (Track, error)
}

// Registry allows provider implementations to be added without coupling the bot
// command layer to a specific music service.
type Registry struct {
	providers []Provider
}

func NewRegistry(providers ...Provider) *Registry {
	return &Registry{providers: append([]Provider(nil), providers...)}
}

func (r *Registry) Providers() []Provider {
	return append([]Provider(nil), r.providers...)
}
