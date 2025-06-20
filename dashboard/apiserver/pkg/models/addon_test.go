package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddonRegistrationSubjectModel(t *testing.T) {
	subject := AddonRegistrationSubject{
		Groups: []string{"system:masters", "admin"},
		User:   "test-user",
	}

	assert.Len(t, subject.Groups, 2)
	assert.Contains(t, subject.Groups, "system:masters")
	assert.Equal(t, "test-user", subject.User)
}

func TestAddonRegistrationModel(t *testing.T) {
	registration := AddonRegistration{
		SignerName: "test-signer",
		Subject: AddonRegistrationSubject{
			Groups: []string{"admin"},
			User:   "test-user",
		},
	}

	assert.Equal(t, "test-signer", registration.SignerName)
	assert.Equal(t, "test-user", registration.Subject.User)
	assert.Len(t, registration.Subject.Groups, 1)
}

func TestAddonSupportedConfigModel(t *testing.T) {
	config := AddonSupportedConfig{
		Group:    "addon.open-cluster-management.io",
		Resource: "addonconfigs",
	}

	assert.Equal(t, "addon.open-cluster-management.io", config.Group)
	assert.Equal(t, "addonconfigs", config.Resource)
}

func TestManagedClusterAddonModel(t *testing.T) {
	addon := ManagedClusterAddon{
		ID:               "test-id",
		Name:             "test-addon",
		Namespace:        "test-cluster",
		InstallNamespace: "open-cluster-management-agent-addon",
		Conditions: []Condition{
			{
				Type:   "Available",
				Status: "True",
			},
		},
		Registrations: []AddonRegistration{
			{
				SignerName: "test-signer",
				Subject: AddonRegistrationSubject{
					User: "test-user",
				},
			},
		},
		SupportedConfigs: []AddonSupportedConfig{
			{
				Group:    "addon.open-cluster-management.io",
				Resource: "addonconfigs",
			},
		},
	}

	assert.Equal(t, "test-id", addon.ID)
	assert.Equal(t, "test-addon", addon.Name)
	assert.Equal(t, "test-cluster", addon.Namespace)
	assert.Equal(t, "open-cluster-management-agent-addon", addon.InstallNamespace)
	assert.Len(t, addon.Conditions, 1)
	assert.Equal(t, "Available", addon.Conditions[0].Type)
	assert.Len(t, addon.Registrations, 1)
	assert.Equal(t, "test-signer", addon.Registrations[0].SignerName)
	assert.Len(t, addon.SupportedConfigs, 1)
	assert.Equal(t, "addon.open-cluster-management.io", addon.SupportedConfigs[0].Group)
}
