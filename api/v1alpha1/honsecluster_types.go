package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
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
    DeploymentMode string                  `json:"deploymentMode,omitempty"`
    Images         HonseImagesSpec         `json:"images"`
    Runtime        HonseRuntimeSpec        `json:"runtime,omitempty"`
    Observability  *HonseObservabilitySpec `json:"observability,omitempty"`
}

type HonseImagesSpec struct {
    Server          string `json:"server"`
    MainFileserver  string `json:"mainFileserver"`
    ShardFileserver string `json:"shardFileserver,omitempty"`
    Adminpanel      string `json:"adminpanel,omitempty"`
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
    Phase         string             `json:"phase,omitempty"`
    LastBuildTime *metav1.Time       `json:"lastBuildTime,omitempty"`
    Conditions    []metav1.Condition `json:"conditions,omitempty"`
}

// Implement runtime.Object for HonseCluster and HonseClusterList
func (in *HonseCluster) DeepCopyObject() runtime.Object {
    if in == nil {
        return nil
    }
    out := new(HonseCluster)
    in.DeepCopyInto(out)
    return out
}

func (in *HonseClusterList) DeepCopyObject() runtime.Object {
    if in == nil {
        return nil
    }
    out := new(HonseClusterList)
    in.DeepCopyInto(out)
    return out
}

func (in *HonseCluster) DeepCopyInto(out *HonseCluster) {
    *out = *in
    out.TypeMeta = in.TypeMeta
    in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
    out.Spec = in.Spec
    if in.Status.LastBuildTime != nil {
        t := *in.Status.LastBuildTime
        out.Status.LastBuildTime = &t
    }
    if in.Status.Conditions != nil {
        out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
        copy(out.Status.Conditions, in.Status.Conditions)
    }
}

func (in *HonseClusterList) DeepCopyInto(out *HonseClusterList) {
    *out = *in
    out.TypeMeta = in.TypeMeta
    in.ListMeta.DeepCopyInto(&out.ListMeta)
    if in.Items != nil {
        out.Items = make([]HonseCluster, len(in.Items))
        for i := range in.Items {
            in.Items[i].DeepCopyInto(&out.Items[i])
        }
    }
}

func init() {
    SchemeBuilder.Register(&HonseCluster{}, &HonseClusterList{})
}
