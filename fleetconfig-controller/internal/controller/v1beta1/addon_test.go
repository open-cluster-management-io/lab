package v1beta1

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

func newAddonClient(t *testing.T) *addonapi.Clientset {
	t.Helper()
	if cfg == nil {
		t.Skip("envtest not available")
	}
	c, err := addonapi.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create addon client: %v", err)
	}
	return c
}

// cleanupMCAs removes all ManagedClusterAddOns in the given namespace.
func cleanupMCAs(t *testing.T, ctx context.Context, c *addonapi.Clientset, ns string) {
	t.Helper()
	list, _ := c.AddonV1alpha1().ManagedClusterAddOns(ns).List(ctx, metav1.ListOptions{})
	if list != nil {
		for _, mca := range list.Items {
			_ = c.AddonV1alpha1().ManagedClusterAddOns(ns).Delete(ctx, mca.Name, metav1.DeleteOptions{})
		}
	}
}

// cleanupADCs removes all AddOnDeploymentConfigs in the given namespace.
func cleanupADCs(t *testing.T, ctx context.Context, c *addonapi.Clientset, ns string) {
	t.Helper()
	list, _ := c.AddonV1alpha1().AddOnDeploymentConfigs(ns).List(ctx, metav1.ListOptions{})
	if list != nil {
		for _, adc := range list.Items {
			_ = c.AddonV1alpha1().AddOnDeploymentConfigs(ns).Delete(ctx, adc.Name, metav1.DeleteOptions{})
		}
	}
}

func ensureNamespace(t *testing.T, ctx context.Context, ns string) {
	t.Helper()
	err := k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
	if err != nil && !kerrs.IsAlreadyExists(err) {
		t.Fatalf("failed to create namespace %s: %v", ns, err)
	}
}

func TestParseMultiDocYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:  "single document",
			input: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n",
			want:  1,
		},
		{
			name:  "multiple documents",
			input: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: b\n",
			want:  2,
		},
		{
			name:  "empty input",
			input: "",
			want:  0,
		},
		{
			name:  "whitespace only",
			input: "   \n\n  \n",
			want:  0,
		},
		{
			name:  "document with trailing separator",
			input: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n---\n",
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMultiDocYAML([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMultiDocYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("parseMultiDocYAML() returned %d manifests, want %d", len(got), tt.want)
			}
		})
	}
}

func TestHandleAddonDisable(t *testing.T) {
	addonC := newAddonClient(t)
	ctx := context.Background()
	ns := "disable-test-ns"
	ensureNamespace(t, ctx, ns)

	t.Cleanup(func() { cleanupMCAs(t, ctx, addonC, ns) })

	// Seed an MCA
	_, err := addonC.AddonV1alpha1().ManagedClusterAddOns(ns).Create(ctx, &addonv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-addon",
			Namespace: ns,
			Labels:    v1beta1.ManagedByLabels,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("seed MCA: %v", err)
	}

	t.Run("deletes existing MCA", func(t *testing.T) {
		err := handleAddonDisable(ctx, addonC, ns, []string{"test-addon"})
		if err != nil {
			t.Fatalf("handleAddonDisable: %v", err)
		}
		_, err = addonC.AddonV1alpha1().ManagedClusterAddOns(ns).Get(ctx, "test-addon", metav1.GetOptions{})
		if err == nil {
			t.Fatal("MCA should have been deleted")
		}
	})

	t.Run("no error on already-deleted addon", func(t *testing.T) {
		err := handleAddonDisable(ctx, addonC, ns, []string{"test-addon"})
		if err != nil {
			t.Fatalf("handleAddonDisable should succeed on NotFound: %v", err)
		}
	})

	t.Run("no error on empty list", func(t *testing.T) {
		err := handleAddonDisable(ctx, addonC, ns, []string{})
		if err != nil {
			t.Fatalf("handleAddonDisable should succeed on empty list: %v", err)
		}
	})
}

func TestHandleAddonEnable(t *testing.T) {
	addonC := newAddonClient(t)
	ctx := context.Background()
	ns := "enable-test-ns"
	ensureNamespace(t, ctx, ns)

	t.Cleanup(func() {
		cleanupMCAs(t, ctx, addonC, ns)
		cleanupADCs(t, ctx, addonC, ns)
	})

	// Seed a CMA so the enable can reference it (upstream would reject unknown addon, but our code doesn't check)
	t.Run("creates MCA with labels and annotations", func(t *testing.T) {
		addons := []v1beta1.AddOn{
			{
				ConfigName: "my-addon",
				Annotations: map[string]string{
					"custom.io/key": "val",
				},
			},
		}
		enabled, err := handleAddonEnable(ctx, addonC, ns, addons)
		if err != nil {
			t.Fatalf("handleAddonEnable: %v", err)
		}
		if len(enabled) != 1 || enabled[0] != "my-addon" {
			t.Fatalf("expected [my-addon], got %v", enabled)
		}

		mca, err := addonC.AddonV1alpha1().ManagedClusterAddOns(ns).Get(ctx, "my-addon", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get MCA: %v", err)
		}
		if mca.Labels[v1beta1.LabelAddOnManagedBy] != "fleetconfig-controller" {
			t.Errorf("expected managedBy label, got %v", mca.Labels)
		}
		if mca.Annotations["custom.io/key"] != "val" {
			t.Errorf("expected custom annotation, got %v", mca.Annotations)
		}
	})

	t.Run("updates MCA on re-enable", func(t *testing.T) {
		addons := []v1beta1.AddOn{
			{
				ConfigName: "my-addon",
				Annotations: map[string]string{
					"custom.io/key": "updated-val",
				},
			},
		}
		enabled, err := handleAddonEnable(ctx, addonC, ns, addons)
		if err != nil {
			t.Fatalf("handleAddonEnable: %v", err)
		}
		if len(enabled) != 1 {
			t.Fatalf("expected 1 enabled, got %d", len(enabled))
		}

		mca, err := addonC.AddonV1alpha1().ManagedClusterAddOns(ns).Get(ctx, "my-addon", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get MCA: %v", err)
		}
		if mca.Annotations["custom.io/key"] != "updated-val" {
			t.Errorf("expected updated annotation, got %v", mca.Annotations)
		}
	})

	t.Run("preserves external annotations on update", func(t *testing.T) {
		cleanupMCAs(t, ctx, addonC, ns)
		cleanupADCs(t, ctx, addonC, ns)

		_, err := addonC.AddonV1alpha1().ManagedClusterAddOns(ns).Create(ctx, &addonv1alpha1.ManagedClusterAddOn{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "overlay-addon",
				Namespace:   ns,
				Labels:      v1beta1.ManagedByLabels,
				Annotations: map[string]string{"third.party/extra": "keep-me"},
			},
			Spec: addonv1alpha1.ManagedClusterAddOnSpec{},
		}, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("seed MCA: %v", err)
		}

		addons := []v1beta1.AddOn{
			{
				ConfigName: "overlay-addon",
				Annotations: map[string]string{
					"spoke.fcc/key": "from-spoke",
				},
			},
		}
		if _, err := handleAddonEnable(ctx, addonC, ns, addons); err != nil {
			t.Fatalf("handleAddonEnable: %v", err)
		}

		mca, err := addonC.AddonV1alpha1().ManagedClusterAddOns(ns).Get(ctx, "overlay-addon", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get MCA: %v", err)
		}
		if mca.Annotations["third.party/extra"] != "keep-me" {
			t.Errorf("external annotation dropped: %v", mca.Annotations)
		}
		if mca.Annotations["spoke.fcc/key"] != "from-spoke" {
			t.Errorf("spoke annotation not set: %v", mca.Annotations)
		}
	})

	t.Run("creates ADC and MCA config ref when DeploymentConfig provided", func(t *testing.T) {
		cleanupMCAs(t, ctx, addonC, ns)
		cleanupADCs(t, ctx, addonC, ns)

		addons := []v1beta1.AddOn{
			{
				ConfigName: "configured-addon",
				DeploymentConfig: &addonv1alpha1.AddOnDeploymentConfigSpec{
					CustomizedVariables: []addonv1alpha1.CustomizedVariable{
						{Name: "IMAGE", Value: "quay.io/test:v1"},
					},
				},
			},
		}
		enabled, err := handleAddonEnable(ctx, addonC, ns, addons)
		if err != nil {
			t.Fatalf("handleAddonEnable: %v", err)
		}
		if len(enabled) != 1 {
			t.Fatalf("expected 1 enabled, got %d", len(enabled))
		}

		// Verify ADC was created
		adc, err := addonC.AddonV1alpha1().AddOnDeploymentConfigs(ns).Get(ctx, "configured-addon", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get ADC: %v", err)
		}
		if adc.Labels[v1beta1.LabelAddOnManagedBy] != "fleetconfig-controller" {
			t.Errorf("ADC missing fleetconfig managed-by label: %v", adc.Labels)
		}
		if len(adc.Spec.CustomizedVariables) != 1 || adc.Spec.CustomizedVariables[0].Value != "quay.io/test:v1" {
			t.Errorf("ADC spec mismatch: %+v", adc.Spec)
		}

		// Verify MCA references it
		mca, err := addonC.AddonV1alpha1().ManagedClusterAddOns(ns).Get(ctx, "configured-addon", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get MCA: %v", err)
		}
		if len(mca.Spec.Configs) != 1 {
			t.Fatalf("expected 1 config ref, got %d", len(mca.Spec.Configs))
		}
		ref := mca.Spec.Configs[0]
		if ref.Name != "configured-addon" || ref.Namespace != ns || ref.Resource != AddOnDeploymentConfigResource {
			t.Errorf("config ref mismatch: %+v", ref)
		}
	})

	t.Run("empty addons list is no-op", func(t *testing.T) {
		enabled, err := handleAddonEnable(ctx, addonC, ns, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled != nil {
			t.Errorf("expected nil, got %v", enabled)
		}
	})
}

func TestHandleSpokeAddons(t *testing.T) {
	addonC := newAddonClient(t)
	ctx := context.Background()
	ns := "spoke-addon-test-ns"
	ensureNamespace(t, ctx, ns)

	t.Cleanup(func() {
		cleanupMCAs(t, ctx, addonC, ns)
		cleanupADCs(t, ctx, addonC, ns)
	})

	t.Run("enables addons from empty state", func(t *testing.T) {
		spoke := &v1beta1.Spoke{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
			Spec: v1beta1.SpokeSpec{
				AddOns: []v1beta1.AddOn{
					{ConfigName: "addon-a"},
					{ConfigName: "addon-b"},
				},
			},
		}
		enabled, err := handleSpokeAddons(ctx, addonC, spoke)
		if err != nil {
			t.Fatalf("handleSpokeAddons: %v", err)
		}
		if len(enabled) != 2 {
			t.Fatalf("expected 2 enabled, got %d: %v", len(enabled), enabled)
		}
	})

	t.Run("disables addons removed from spec", func(t *testing.T) {
		spoke := &v1beta1.Spoke{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
			Spec: v1beta1.SpokeSpec{
				AddOns: []v1beta1.AddOn{
					{ConfigName: "addon-a"},
				},
			},
		}
		enabled, err := handleSpokeAddons(ctx, addonC, spoke)
		if err != nil {
			t.Fatalf("handleSpokeAddons: %v", err)
		}
		if len(enabled) != 1 || enabled[0] != "addon-a" {
			t.Fatalf("expected [addon-a], got %v", enabled)
		}

		// addon-b should be deleted
		_, err = addonC.AddonV1alpha1().ManagedClusterAddOns(ns).Get(ctx, "addon-b", metav1.GetOptions{})
		if err == nil {
			t.Error("addon-b MCA should have been deleted")
		}
	})

	t.Run("empty spec disables all", func(t *testing.T) {
		spoke := &v1beta1.Spoke{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
			Spec:       v1beta1.SpokeSpec{},
		}
		enabled, err := handleSpokeAddons(ctx, addonC, spoke)
		if err != nil {
			t.Fatalf("handleSpokeAddons: %v", err)
		}
		if len(enabled) != 0 {
			t.Fatalf("expected 0 enabled, got %v", enabled)
		}
	})
}
