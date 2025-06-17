package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// DockerNetwork represents a subset of the JSON structure from `docker network inspect`
type DockerNetwork struct {
	Containers map[string]struct {
		Name        string `json:"Name"`
		IPv4Address string `json:"IPv4Address"`
	} `json:"Containers"`
}

// GetKindNodeIP extracts a kind cluster's node IP from the kind docker network by cluster name
func GetKindNodeIP(clusterName string) (string, error) {
	cmd := exec.Command("docker", "network", "inspect", "kind")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to inspect docker network: %w", err)
	}

	var networks []DockerNetwork
	if err := json.Unmarshal(out.Bytes(), &networks); err != nil {
		return "", fmt.Errorf("failed to parse docker network JSON: %w", err)
	}

	if len(networks) > 0 && len(networks[0].Containers) > 0 {
		for _, c := range networks[0].Containers {
			if strings.HasPrefix(c.Name, clusterName) {
				return strings.Split(c.IPv4Address, "/")[0], nil
			}
		}
	}

	return "", fmt.Errorf("no container found in Docker network for cluster %s", clusterName)
}
