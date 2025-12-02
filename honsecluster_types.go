package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type HonseCluster struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   HonseClusterSpec   `json:"spec,omitempty"`
    Status HonseClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type HonseClusterList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []HonseCluster `json:"items"`
}

type HonseClusterSpec struct {
    Version       string                 `json:"version,omitempty"`
    Source        HonseSourceSpec        `json:"source"`
    Build         HonseBuildSpec         `json:"build,omitempty"`
    Registry      HonseRegistrySpec      `json:"registry"`
    Runtime       HonseRuntimeSpec       `json:"runtime,omitempty"`
    Observability *HonseObservabilitySpec `json:"observability,omitempty"`
}

type HonseSourceSpec struct {
    RepoURL        string `json:"repoUrl"`
    Ref            string `json:"ref,omitempty"`
    ContextBaseDir string `json:"contextBaseDir,omitempty"`
}

type HonseBuildSpec struct {
    Strategy   string                    `json:"strategy,omitempty"`
    Components []HonseBuildComponentSpec `json:"components,omitempty"`
}

type HonseBuildComponentSpec struct {
    Name       string `json:"name"`
    ContextDir string `json:"contextDir"`
    Dockerfile string `json:"dockerfile"`
}

type HonseRegistrySpec struct {
    Host             string `json:"host"`
    RepositoryPrefix string `json:"repositoryPrefix,omitempty"`
    Insecure         bool   `json:"insecure,omitempty"`
    SecretRef        string `json:"secretRef,omitempty"`
}

type HonseRuntimeSpec struct {
    IngressHost             string `json:"ingressHost,omitempty"`
    IngressClassName        string `json:"ingressClassName,omitempty"`
    IngressTLSSecretRef     string `json:"ingressTlsSecretRef,omitempty"`
    StorageClass            string `json:"storageClass,omitempty"`
    StorageSize             string `json:"storageSize,omitempty"`
    FederationServerType    string `json:"federationServerType,omitempty"`
    FederationJoinSecretRef string `json:"federationJoinSecretRef,omitempty"`
}

type HonseObservabilitySpec struct {
    Enabled               bool `json:"enabled,omitempty"`
    MetricsServiceMonitor bool `json:"metricsServiceMonitor,omitempty"`
}

type HonseClusterStatus struct {
    Phase           string             `json:"phase,omitempty"`
    LastBuildCommit string             `json:"lastBuildCommit,omitempty"`
    LastBuildTime   *metav1.Time       `json:"lastBuildTime,omitempty"`
    Conditions      []metav1.Condition `json:"conditions,omitempty"`
}

func init() {
    SchemeBuilder.Register(&HonseCluster{}, &HonseClusterList{})
}
