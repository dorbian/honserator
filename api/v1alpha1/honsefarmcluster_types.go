package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HonseFarmClusterSpec defines the desired state of HonseFarmCluster
type HonseFarmClusterSpec struct {
    // Namespace where HonseFarm components should be deployed
    Namespace string `json:"namespace"`

    // Base DNS name used for the cluster entrypoint (e.g. cluster.honse.farm)
    APIDomain string `json:"apiDomain"`

    // Container registry (e.g. ghcr.io/dorbian)
    Registry string `json:"registry,omitempty"`

    // Images for the HonseFarm services
    Images HonseFarmImages `json:"images,omitempty"`

    // Cloudflared configuration for exposing services externally
    Cloudflared *CloudflaredSpec `json:"cloudflared,omitempty"`
}

type HonseFarmImages struct {
    Server        string `json:"server,omitempty"`
    MainFileserver string `json:"mainFileserver,omitempty"`
    ShardFileserver string `json:"shardFileserver,omitempty"`
    AdminPanel    string `json:"adminPanel,omitempty"`
    Observability string `json:"observability,omitempty"`
}

// CloudflaredSpec configures a Cloudflare tunnel
type CloudflaredSpec struct {
    Enabled bool `json:"enabled"`

    // Cloudflared image to use
    Image string `json:"image,omitempty"`

    // Human-readable name of the tunnel
    TunnelName string `json:"tunnelName,omitempty"`

    // UUID of the tunnel
    TunnelID string `json:"tunnelId,omitempty"`

    // Reference to credentials secret
    CredentialsSecretRef *SecretRef `json:"credentialsSecretRef,omitempty"`

    // Additional CLI args
    ExtraArgs []string `json:"extraArgs,omitempty"`

    // Ingress rules mapping hostnames to services
    Ingress []CloudflaredIngressRule `json:"ingress,omitempty"`
}

type SecretRef struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"`
}

type CloudflaredIngressRule struct {
    Hostname         string `json:"hostname,omitempty"`
    ServiceName      string `json:"serviceName,omitempty"`
    ServiceNamespace string `json:"serviceNamespace,omitempty"`
    ServicePort      int32  `json:"servicePort,omitempty"`
    SpecialService   string `json:"specialService,omitempty"`
}

// HonseFarmClusterStatus defines the observed state of HonseFarmCluster
type HonseFarmClusterStatus struct {
    Phase      string              `json:"phase,omitempty"`
    Conditions []metav1.Condition  `json:"conditions,omitempty"`
    CloudflaredStatus *CloudflaredStatus `json:"cloudflaredStatus,omitempty"`
}

type CloudflaredStatus struct {
    Ready     bool   `json:"ready,omitempty"`
    LastError string `json:"lastError,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HonseFarmCluster is the Schema for the honsefarmclusters API
type HonseFarmCluster struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   HonseFarmClusterSpec   `json:"spec,omitempty"`
    Status HonseFarmClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HonseFarmClusterList contains a list of HonseFarmCluster
type HonseFarmClusterList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []HonseFarmCluster `json:"items"`
}

func init() {
    SchemeBuilder.Register(&HonseFarmCluster{}, &HonseFarmClusterList{})
}
