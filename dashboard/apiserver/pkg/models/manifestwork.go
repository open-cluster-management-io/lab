package models

// ManifestWork represents a simplified version of the OCM ManifestWork resource
type ManifestWork struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Namespace         string                 `json:"namespace"`
	Labels            map[string]string      `json:"labels,omitempty"`
	Manifests         []Manifest             `json:"manifests,omitempty"`
	Conditions        []Condition            `json:"conditions,omitempty"`
	ResourceStatus    ManifestResourceStatus `json:"resourceStatus,omitempty"`
	CreationTimestamp string                 `json:"creationTimestamp,omitempty"`
}

// Manifest represents a resource to be deployed on a managed cluster
type Manifest struct {
	RawExtension map[string]interface{} `json:"rawExtension,omitempty"`
}

// ManifestResourceStatus represents the status of resources in a manifest work
type ManifestResourceStatus struct {
	Manifests []ManifestCondition `json:"manifests,omitempty"`
}

// ManifestCondition represents the conditions of resources deployed on a managed cluster
type ManifestCondition struct {
	ResourceMeta ManifestResourceMeta `json:"resourceMeta"`
	Conditions   []Condition          `json:"conditions"`
}

// ManifestResourceMeta represents the metadata of a resource in a manifest
type ManifestResourceMeta struct {
	Ordinal   int32  `json:"ordinal"`
	Group     string `json:"group,omitempty"`
	Version   string `json:"version,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Resource  string `json:"resource,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// DeleteOption represents the deletion strategy when the manifestwork is deleted
type DeleteOption struct {
	PropagationPolicy string             `json:"propagationPolicy"`
	SelectivelyOrphan *SelectivelyOrphan `json:"selectivelyOrphans,omitempty"`
}

// SelectivelyOrphan represents resources following orphan deletion strategy
type SelectivelyOrphan struct {
	OrphaningRules []OrphaningRule `json:"orphaningRules,omitempty"`
}

// OrphaningRule identifies a single resource to be orphaned
type OrphaningRule struct {
	Group     string `json:"group,omitempty"`
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// ManifestWorkSpec represents the desired configuration of manifests to be deployed
type ManifestWorkSpec struct {
	Workload        []Manifest             `json:"workload,omitempty"`
	DeleteOption    *DeleteOption          `json:"deleteOption,omitempty"`
	ManifestConfigs []ManifestConfigOption `json:"manifestConfigs,omitempty"`
}

// ManifestConfigOption represents the configurations of a manifest
type ManifestConfigOption struct {
	ResourceIdentifier ResourceIdentifier `json:"resourceIdentifier"`
	FeedbackRules      []FeedbackRule     `json:"feedbackRules,omitempty"`
	UpdateStrategy     *UpdateStrategy    `json:"updateStrategy,omitempty"`
}

// ResourceIdentifier identifies a single resource
type ResourceIdentifier struct {
	Group     string `json:"group,omitempty"`
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// UpdateStrategy defines the strategy to update a manifest
type UpdateStrategy struct {
	Type            string                 `json:"type,omitempty"`
	ServerSideApply *ServerSideApplyConfig `json:"serverSideApply,omitempty"`
}

// ServerSideApplyConfig defines the server-side apply configuration
type ServerSideApplyConfig struct {
	Force        bool          `json:"force"`
	FieldManager string        `json:"fieldManager,omitempty"`
	IgnoreFields []IgnoreField `json:"ignoreFields,omitempty"`
}

// IgnoreField defines the fields to be ignored during apply
type IgnoreField struct {
	Condition string   `json:"condition"`
	JSONPaths []string `json:"jsonPaths"`
}

// FeedbackRule defines how status can be returned
type FeedbackRule struct {
	Type      string     `json:"type"`
	JsonPaths []JsonPath `json:"jsonPaths,omitempty"`
}

// JsonPath defines the json path of a field under status
type JsonPath struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Path    string `json:"path"`
}

// ManifestWorkList represents a list of ManifestWork objects
type ManifestWorkList struct {
	Items []ManifestWork `json:"items"`
}
