// Package watch contains a generic watcher that implements manager.Runnable
package watch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultTimeout = 10 * time.Second

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
	timeout   time.Duration
	name      string
	condition ConditionFunc
	handler   HandlerFunc
}

// Config for creating a new ResourceWatcher
type Config struct {
	Client    client.Client
	Log       logr.Logger
	Interval  time.Duration
	Timeout   time.Duration
	Name      string
	Condition ConditionFunc
	Handler   HandlerFunc
}

// New creates a new ResourceWatcher. Returns an error if misconfigured.
func New(cfg Config) (*ResourceWatcher, error) {
	if cfg.Client == nil {
		return nil, errors.New("watch.Config.Client must not be nil")
	}
	if cfg.Log.GetSink() == nil {
		return nil, errors.New("watch.Config.Log must not be nil")
	}
	if cfg.Condition == nil {
		return nil, errors.New("watch.Config.Condition must not be nil")
	}
	if cfg.Handler == nil {
		return nil, errors.New("watch.Config.Handler must not be nil")
	}
	if cfg.Interval <= 0 {
		return nil, errors.New("watch.Config.Interval must be positive")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &ResourceWatcher{
		client:    cfg.Client,
		log:       cfg.Log,
		interval:  cfg.Interval,
		timeout:   timeout,
		name:      cfg.Name,
		condition: cfg.Condition,
		handler:   cfg.Handler,
	}, nil
}

// NewOrDie creates a new ResourceWatcher. Panics if misconfigured.
func NewOrDie(cfg Config) *ResourceWatcher {
	rw, err := New(cfg)
	if err != nil {
		panic(fmt.Errorf("failed to create ResourceWatcher due to invalid input: %w", err))
	}
	return rw
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
			func() {
				defer func() {
					if r := recover(); r != nil {
						w.log.Error(fmt.Errorf("panic: %v", r), "Watch check panicked", "name", w.name)
					}
				}()
				if err := w.check(ctx); err != nil {
					w.log.Error(err, "Watch check failed", "name", w.name)
				}
			}()
		}
	}
}

func (w *ResourceWatcher) check(ctx context.Context) error {
	cCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	met, err := w.condition(cCtx, w.client)
	if err != nil {
		return err
	}

	if !met {
		return nil
	}

	w.log.V(1).Info("Condition met, executing handler", "name", w.name)
	hCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	return w.handler(hCtx, w.client)
}
