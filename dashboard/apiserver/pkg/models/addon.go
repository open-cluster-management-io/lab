package models

// AddonRegistrationSubject represents the subject of a registration for an addon
type AddonRegistrationSubject struct {
	Groups []string `json:"groups"`
	User   string   `json:"user"`
}

// AddonRegistration represents the registration information for an addon
type AddonRegistration struct {
	SignerName string                   `json:"signerName"`
	Subject    AddonRegistrationSubject `json:"subject"`
}

// AddonSupportedConfig represents a supported configuration for an addon
type AddonSupportedConfig struct {
	Group    string `json:"group"`
	Resource string `json:"resource"`
}

// ManagedClusterAddon represents a simplified OCM ManagedClusterAddOn
type ManagedClusterAddon struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Namespace         string                 `json:"namespace"`
	InstallNamespace  string                 `json:"installNamespace"`
	CreationTimestamp string                 `json:"creationTimestamp,omitempty"`
	Conditions        []Condition            `json:"conditions,omitempty"`
	Registrations     []AddonRegistration    `json:"registrations,omitempty"`
	SupportedConfigs  []AddonSupportedConfig `json:"supportedConfigs,omitempty"`
}
