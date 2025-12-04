package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
)

type HonseFarmClusterSpec struct {
    Namespace    string            `json:"namespace"`
    APIDomain    string            `json:"apiDomain"`
    Hosts        *HostsSpec        `json:"hosts,omitempty"`
    Global       *GlobalConfig     `json:"global,omitempty"`
    Images       *ImagesSpec       `json:"images,omitempty"`
    Components   *ComponentsSpec   `json:"components,omitempty"`
    Certificates *CertificatesSpec `json:"certificates,omitempty"`
    Cloudflared  *CloudflaredSpec  `json:"cloudflared,omitempty"`
}

type HostsSpec struct {
    Server string      `json:"server,omitempty"`
    Admin  string      `json:"admin,omitempty"`
    CDN    string      `json:"cdn,omitempty"`
    Shards []HostShard `json:"shards,omitempty"`
}

type HostShard struct {
    Name string `json:"name,omitempty"`
    Host string `json:"host,omitempty"`
}

type GlobalConfig struct {
    Logging    *GlobalLogging    `json:"logging,omitempty"`
    Database   *GlobalDatabase   `json:"database,omitempty"`
    Redis      *GlobalRedis      `json:"redis,omitempty"`
    JWT        *GlobalJWT        `json:"jwt,omitempty"`
    Telemetry  *GlobalTelemetry  `json:"telemetry,omitempty"`
    Federation *GlobalFederation `json:"federation,omitempty"`
}

type GlobalLogging struct {
    DefaultLevel    string `json:"defaultLevel,omitempty"`
    MicrosoftLevel  string `json:"microsoftLevel,omitempty"`
    AspNetCoreLevel string `json:"aspNetCoreLevel,omitempty"`
}

type GlobalDatabase struct {
    Host     string `json:"host,omitempty"`
    Name     string `json:"name,omitempty"`
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
}

type GlobalRedis struct {
    ConnectionString string `json:"connectionString,omitempty"`
    Pool             int32  `json:"pool,omitempty"`
}

type GlobalJWT struct {
    Secret string `json:"secret,omitempty"`
}

type GlobalTelemetry struct {
    LogsEndpoint              string `json:"logsEndpoint,omitempty"`
    AnalyticsOptIn            bool   `json:"analyticsOptIn,omitempty"`
    AnalyticsConnectionString string `json:"analyticsConnectionString,omitempty"`
}

type GlobalFederation struct {
    ServerID             string `json:"serverId,omitempty"`
    ServerName           string `json:"serverName,omitempty"`
    ServerDescription    string `json:"serverDescription,omitempty"`
    ServerVersion        string `json:"serverVersion,omitempty"`
    ServerLocation       string `json:"serverLocation,omitempty"`
    ServerDiscordLink    string `json:"serverDiscordLink,omitempty"`
    ServerType           string `json:"serverType,omitempty"`
    ServerJoinSecret     string `json:"serverJoinSecret,omitempty"`
    ServerBaseURL        string `json:"serverBaseUrl,omitempty"`
    UseDNSBootstrap      bool   `json:"useDnsBootstrap,omitempty"`
    DNSBootstrapHostname string `json:"dnsBootstrapHostname,omitempty"`
    GroupUIDPrefix       string `json:"groupUidPrefix,omitempty"`
    Role                 string `json:"role,omitempty"`
}

type ImagesSpec struct {
    Server          string `json:"server,omitempty"`
    AdminPanel      string `json:"adminPanel,omitempty"`
    MainFileserver  string `json:"mainFileserver,omitempty"`
    ShardFileserver string `json:"shardFileserver,omitempty"`
}

type ComponentsSpec struct {
    Server      *ServerComponentSpec     `json:"server,omitempty"`
    AdminPanel  *AdminPanelComponentSpec `json:"adminPanel,omitempty"`
    Fileservers *FileserversSpec         `json:"fileservers,omitempty"`
}

type StorageSpec struct {
    Size             string   `json:"size,omitempty"`
    StorageClassName string   `json:"storageClassName,omitempty"`
    AccessModes      []string `json:"accessModes,omitempty"`
}

type ServerComponentSpec struct {
    Replicas        *int32               `json:"replicas,omitempty"`
    Storage         *StorageSpec         `json:"storage,omitempty"`
    ConfigOverrides *runtime.RawExtension `json:"configOverrides,omitempty"`
}

type AdminPanelComponentSpec struct {
    Replicas        *int32               `json:"replicas,omitempty"`
    Storage         *StorageSpec         `json:"storage,omitempty"`
    ConfigOverrides *runtime.RawExtension `json:"configOverrides,omitempty"`
}

type FileserversSpec struct {
    Main   *MainFileserverSpec `json:"main,omitempty"`
    Shards []ShardSpec         `json:"shards,omitempty"`
}

type MainFileserverSpec struct {
    Replicas        *int32               `json:"replicas,omitempty"`
    Storage         *StorageSpec         `json:"storage,omitempty"`
    ConfigOverrides *runtime.RawExtension `json:"configOverrides,omitempty"`
}

type ShardSpec struct {
    Name           string                `json:"name"`
    ReplicaProfile string                `json:"replicaProfile,omitempty"`
    Replicas       *int32               `json:"replicas,omitempty"`
    Storage        *StorageSpec         `json:"storage,omitempty"`
    ConfigOverrides *runtime.RawExtension `json:"configOverrides,omitempty"`
}

type CertificatesSpec struct {
    Mode      string     `json:"mode,omitempty"`
    IssuerRef *IssuerRef `json:"issuerRef,omitempty"`
    DNSNames  []string   `json:"dnsNames,omitempty"`
}

type IssuerRef struct {
    Name string `json:"name,omitempty"`
    Kind string `json:"kind,omitempty"`
}

type CloudflaredSpec struct {
    Enabled              bool                     `json:"enabled,omitempty"`
    Image                string                   `json:"image,omitempty"`
    TunnelName           string                   `json:"tunnelName,omitempty"`
    TunnelID             string                   `json:"tunnelId,omitempty"`
    CredentialsSecretRef *SecretRef               `json:"credentialsSecretRef,omitempty"`
    ExtraArgs            []string                 `json:"extraArgs,omitempty"`
    Ingress              []CloudflaredIngressRule `json:"ingress,omitempty"`
}

type SecretRef struct {
    Name      string `json:"name,omitempty"`
    Namespace string `json:"namespace,omitempty"`
}

type CloudflaredIngressRule struct {
    Hostname         string `json:"hostname,omitempty"`
    Component        string `json:"component,omitempty"`
    ShardName        string `json:"shardName,omitempty"`
    ServiceName      string `json:"serviceName,omitempty"`
    ServiceNamespace string `json:"serviceNamespace,omitempty"`
    ServicePort      int32  `json:"servicePort,omitempty"`
    SpecialService   string `json:"specialService,omitempty"`
}

type HonseFarmClusterStatus struct {
    Phase             string             `json:"phase,omitempty"`
    CloudflaredStatus *CloudflaredStatus `json:"cloudflaredStatus,omitempty"`
}

type CloudflaredStatus struct {
    Ready     bool   `json:"ready,omitempty"`
    LastError string `json:"lastError,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type HonseFarmCluster struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   HonseFarmClusterSpec   `json:"spec,omitempty"`
    Status HonseFarmClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type HonseFarmClusterList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []HonseFarmCluster `json:"items"`
}

func (in *HonseFarmCluster) DeepCopyInto(out *HonseFarmCluster) {
    *out = *in
    out.TypeMeta = in.TypeMeta
    in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
    out.Spec = in.Spec
    out.Status = in.Status
}

func (in *HonseFarmCluster) DeepCopy() *HonseFarmCluster {
    if in == nil {
        return nil
    }
    out := new(HonseFarmCluster)
    in.DeepCopyInto(out)
    return out
}

func (in *HonseFarmCluster) DeepCopyObject() runtime.Object {
    if c := in.DeepCopy(); c != nil {
        return c
    }
    return nil
}

func (in *HonseFarmClusterList) DeepCopyInto(out *HonseFarmClusterList) {
    *out = *in
    out.TypeMeta = in.TypeMeta
    in.ListMeta.DeepCopyInto(&out.ListMeta)
    if in.Items != nil {
        out.Items = make([]HonseFarmCluster, len(in.Items))
        for i := range in.Items {
            in.Items[i].DeepCopyInto(&out.Items[i])
        }
    }
}

func (in *HonseFarmClusterList) DeepCopy() *HonseFarmClusterList {
    if in == nil {
        return nil
    }
    out := new(HonseFarmClusterList)
    in.DeepCopyInto(out)
    return out
}

func (in *HonseFarmClusterList) DeepCopyObject() runtime.Object {
    if c := in.DeepCopy(); c != nil {
        return c
    }
    return nil
}

func init() {
    SchemeBuilder.Register(&HonseFarmCluster{}, &HonseFarmClusterList{})
}
