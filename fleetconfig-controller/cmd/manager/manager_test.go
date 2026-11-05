package manager

import (
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

func TestResolveTokenFile(t *testing.T) {
	const basePath = "/managed/hub-kubeconfig"

	tests := []struct {
		name       string
		kubeconfig string
		want       string
	}{
		{
			name: "relative token file",
			kubeconfig: `apiVersion: v1
kind: Config
clusters:
- name: hub
  cluster:
    server: https://hub.example
users:
- name: hub
  user:
    tokenFile: token
contexts:
- name: hub
  context:
    cluster: hub
    user: hub
current-context: hub
`,
			want: "/managed/hub-kubeconfig/token",
		},
		{
			name: "absolute token file",
			kubeconfig: `apiVersion: v1
kind: Config
clusters:
- name: hub
  cluster:
    server: https://hub.example
users:
- name: hub
  user:
    tokenFile: /var/run/token
contexts:
- name: hub
  context:
    cluster: hub
    user: hub
current-context: hub
`,
			want: "/var/run/token",
		},
		{
			name: "no token file",
			kubeconfig: `apiVersion: v1
kind: Config
clusters:
- name: hub
  cluster:
    server: https://hub.example
users:
- name: hub
  user:
    client-certificate: tls.crt
    client-key: tls.key
contexts:
- name: hub
  context:
    cluster: hub
    user: hub
current-context: hub
`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := resolveTokenFile([]byte(tt.kubeconfig), basePath)
			if err != nil {
				t.Fatalf("resolveTokenFile() error = %v", err)
			}

			config, err := clientcmd.Load(out)
			if err != nil {
				t.Fatalf("clientcmd.Load() error = %v", err)
			}

			got := ""
			if authInfo, ok := config.AuthInfos["hub"]; ok {
				got = authInfo.TokenFile
			}
			if got != tt.want {
				t.Errorf("tokenFile = %q, want %q", got, tt.want)
			}
		})
	}
}
