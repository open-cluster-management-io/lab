//go:build tools
// +build tools

// Package dependencymagnet is used to vendor implicit go dependencies.
package dependencymagnet

import (
	_ "github.com/openshift/build-machinery-go"
	_ "github.com/openshift/imagebuilder/cmd/imagebuilder"
)
