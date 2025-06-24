package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	tests := []struct {
		name        string
		debugMode   string
		useMock     string
		expectPanic bool
	}{
		{
			name:        "debug mode enabled",
			debugMode:   "true",
			useMock:     "true",
			expectPanic: false,
		},
		{
			name:        "debug mode disabled",
			debugMode:   "false",
			useMock:     "true",
			expectPanic: false,
		},
		{
			name:        "no environment variables",
			debugMode:   "",
			useMock:     "true",
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DASHBOARD_DEBUG", tt.debugMode)
			os.Setenv("DASHBOARD_USE_MOCK", tt.useMock)
			defer func() {
				os.Unsetenv("DASHBOARD_DEBUG")
				os.Unsetenv("DASHBOARD_USE_MOCK")
			}()

			if tt.expectPanic {
				assert.Panics(t, func() {
					main()
				})
			} else {
				assert.NotPanics(t, func() {
					defer func() {
						if r := recover(); r != nil {
							if r != "test exit" {
								panic(r)
							}
						}
					}()
					main()
				})
			}
		})
	}
}
