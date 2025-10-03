// Package watch contains a generic watcher that implements manager.Runnable
package watch

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConditionFunc checks if a condition is met
// Returns (conditionMet, error)
type ConditionFunc func(ctx context.Context, c client.Client) (bool, error)

// HandlerFunc is called when the condition is met
type HandlerFunc func(ctx context.Context, c client.Client) error

// ResourceWatcher periodically checks a condition and triggers a handler
type ResourceWatcher struct {
	client    client.Client
	log       logr.Logger
	interval  time.Duration
	name      string
	condition ConditionFunc
	handler   HandlerFunc
}

// Config for creating a new ResourceWatcher
type Config struct {
	Client    client.Client
	Log       logr.Logger
	Interval  time.Duration
	Name      string
	Condition ConditionFunc
	Handler   HandlerFunc
}

// New creates a new ResourceWatcher
func New(cfg Config) *ResourceWatcher {
	return &ResourceWatcher{
		client:    cfg.Client,
		log:       cfg.Log,
		interval:  cfg.Interval,
		name:      cfg.Name,
		condition: cfg.Condition,
		handler:   cfg.Handler,
	}
}

// Start begins the watch loop
func (w *ResourceWatcher) Start(ctx context.Context) error {
	w.log.Info("Starting resource watcher", "name", w.name, "watchInterval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Shutting down resource watcher", "name", w.name)
			return nil
		case <-ticker.C:
			if err := w.check(ctx); err != nil {
				w.log.Error(err, "Watch check failed", "name", w.name)
			}
		}
	}
}

func (w *ResourceWatcher) check(ctx context.Context) error {
	met, err := w.condition(ctx, w.client)
	if err != nil {
		return err
	}

	if !met {
		return nil
	}

	w.log.V(1).Info("Condition met, executing handler", "name", w.name)
	return w.handler(ctx, w.client)
}
