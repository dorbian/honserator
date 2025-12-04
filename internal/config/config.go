package config

import (
    "encoding/json"
    "fmt"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    v1alpha1 "honsefarm-operator/api/v1alpha1"
)

// BuildConfigMap builds a ConfigMap containing the core HonseFarm appsettings
// JSON files for server, adminpanel, main-fileserver and shards.
//
// Keys:
// - server.appsettings.Production.json
// - adminpanel.appsettings.Production.json
// - main-fileserver.appsettings.Production.json
// - shard-<name>.appsettings.Production.json
func BuildConfigMap(cluster *v1alpha1.HonseFarmCluster) (*corev1.ConfigMap, error) {
    ns := cluster.Spec.Namespace
    if ns == "" {
        ns = "honsefarm"
    }

    data := map[string]string{}

    // Server config
    serverCfg := buildServerConfig(cluster)
    if cluster.Spec.Components != nil && cluster.Spec.Components.Server != nil && cluster.Spec.Components.Server.ConfigOverrides != nil && len(cluster.Spec.Components.Server.ConfigOverrides.Raw) > 0 {
        serverCfg = mergeOverride(serverCfg, cluster.Spec.Components.Server.ConfigOverrides.Raw)
    }
    if b, err := json.Marshal(serverCfg); err == nil {
        data["server.appsettings.Production.json"] = string(b)
    } else {
        return nil, fmt.Errorf("marshal server config: %w", err)
    }

    // Admin panel config
    adminCfg := buildAdminConfig(cluster)
    if cluster.Spec.Components != nil && cluster.Spec.Components.AdminPanel != nil && cluster.Spec.Components.AdminPanel.ConfigOverrides != nil && len(cluster.Spec.Components.AdminPanel.ConfigOverrides.Raw) > 0 {
        adminCfg = mergeOverride(adminCfg, cluster.Spec.Components.AdminPanel.ConfigOverrides.Raw)
    }
    if b, err := json.Marshal(adminCfg); err == nil {
        data["adminpanel.appsettings.Production.json"] = string(b)
    } else {
        return nil, fmt.Errorf("marshal adminpanel config: %w", err)
    }

    // Main fileserver config
    if cluster.Spec.Components != nil && cluster.Spec.Components.Fileservers != nil && cluster.Spec.Components.Fileservers.Main != nil {
        mainCfg := buildMainFileserverConfig(cluster)
        if cluster.Spec.Components.Fileservers.Main.ConfigOverrides != nil && len(cluster.Spec.Components.Fileservers.Main.ConfigOverrides.Raw) > 0 {
            mainCfg = mergeOverride(mainCfg, cluster.Spec.Components.Fileservers.Main.ConfigOverrides.Raw)
        }
        if b, err := json.Marshal(mainCfg); err == nil {
            data["main-fileserver.appsettings.Production.json"] = string(b)
        } else {
            return nil, fmt.Errorf("marshal main-fileserver config: %w", err)
        }
    }

    // Shard configs
    if cluster.Spec.Components != nil && cluster.Spec.Components.Fileservers != nil {
        for _, shard := range cluster.Spec.Components.Fileservers.Shards {
            shardCfg := buildShardFileserverConfig(cluster, &shard)
            if shard.ConfigOverrides != nil && len(shard.ConfigOverrides.Raw) > 0 {
                shardCfg = mergeOverride(shardCfg, shard.ConfigOverrides.Raw)
            }
            if b, err := json.Marshal(shardCfg); err == nil {
                key := fmt.Sprintf("%s.appsettings.Production.json", shard.Name)
                data[key] = string(b)
            } else {
                return nil, fmt.Errorf("marshal shard config %s: %w", shard.Name, err)
            }
        }
    }

    cm := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "honsefarm-config",
            Namespace: ns,
            Labels: map[string]string{
                "app.kubernetes.io/managed-by": "honsefarm-operator",
                "app.kubernetes.io/name":       "honsefarm-config",
            },
        },
        Data: data,
    }

    return cm, nil
}

// Shallow merge of override JSON onto base map.
func mergeOverride(base map[string]interface{}, raw []byte) map[string]interface{} {
    if len(raw) == 0 {
        return base
    }
    var override map[string]interface{}
    if err := json.Unmarshal(raw, &override); err != nil {
        // If override is not an object, ignore it.
        return base
    }
    for k, v := range override {
        base[k] = v
    }
    return base
}

func buildServerConfig(cluster *v1alpha1.HonseFarmCluster) map[string]interface{} {
    cfg := map[string]interface{}{}

    logging := map[string]interface{}{}
    if cluster.Spec.Global != nil && cluster.Spec.Global.Logging != nil {
        lg := cluster.Spec.Global.Logging
        if lg.DefaultLevel != "" {
            logging["Default"] = lg.DefaultLevel
        }
        if lg.MicrosoftLevel != "" {
            logging["Microsoft"] = lg.MicrosoftLevel
        }
        if lg.AspNetCoreLevel != "" {
            logging["MicrosoftHostingLifetime"] = lg.AspNetCoreLevel
        }
    }
    if len(logging) > 0 {
        cfg["Logging"] = map[string]interface{}{
            "LogLevel": logging,
        }
    }

    // Connection string
    if cluster.Spec.Global != nil && cluster.Spec.Global.Database != nil {
        db := cluster.Spec.Global.Database
        conn := fmt.Sprintf("Host=%s;Database=%s;Username=%s;Password=%s", db.Host, db.Name, db.Username, db.Password)
        cfg["ConnectionStrings"] = map[string]interface{}{
            "Database": conn,
        }
    }

    cfg["AllowedHosts"] = "*"

    // Federation
    if cluster.Spec.Global != nil && cluster.Spec.Global.Federation != nil {
        f := cluster.Spec.Global.Federation
        fed := map[string]interface{}{}
        if f.ServerID != "" {
            fed["ServerId"] = f.ServerID
        }
        if f.ServerName != "" {
            fed["ServerName"] = f.ServerName
        }
        if f.ServerDescription != "" {
            fed["ServerDescription"] = f.ServerDescription
        }
        if f.ServerVersion != "" {
            fed["ServerVersion"] = f.ServerVersion
        }
        if f.ServerLocation != "" {
            fed["ServerLocation"] = f.ServerLocation
        }
        if f.ServerDiscordLink != "" {
            fed["ServerDiscordLink"] = f.ServerDiscordLink
        }
        if f.ServerType != "" {
            fed["ServerType"] = f.ServerType
        }
        if f.ServerJoinSecret != "" {
            fed["ServerJoinSecret"] = f.ServerJoinSecret
        }
        if f.ServerBaseURL != "" {
            fed["ServerBaseUrl"] = f.ServerBaseURL
        }
        fed["UseDnsBootstrap"] = f.UseDNSBootstrap
        if f.DNSBootstrapHostname != "" {
            fed["DnsBootstrapHostname"] = f.DNSBootstrapHostname
        }
        if f.GroupUIDPrefix != "" {
            fed["GroupUidPrefix"] = f.GroupUIDPrefix
        }
        if f.Role != "" {
            fed["Role"] = f.Role
        }
        cfg["Federation"] = fed
    }

    // HonseFarm core
    hf := map[string]interface{}{}
    if cluster.Spec.Global != nil {
        if cluster.Spec.Global.JWT != nil && cluster.Spec.Global.JWT.Secret != "" {
            hf["Jwt"] = cluster.Spec.Global.JWT.Secret
        }
        if cluster.Spec.Global.Redis != nil {
            hf["RedisConnectionString"] = cluster.Spec.Global.Redis.ConnectionString
            if cluster.Spec.Global.Redis.Pool != 0 {
                hf["RedisPool"] = cluster.Spec.Global.Redis.Pool
            }
        }
        if cluster.Spec.Global.Telemetry != nil {
            t := cluster.Spec.Global.Telemetry
            if t.LogsEndpoint != "" {
                hf["OpenTelemetryLogsEndpoint"] = t.LogsEndpoint
            }
            hf["OpenTelemetryAnalyticsOptIn"] = t.AnalyticsOptIn
            if t.AnalyticsConnectionString != "" {
                hf["OpenTelemetryAnalyticsConnectionString"] = t.AnalyticsConnectionString
            }
        }
    }
    // Reasonable defaults mirroring your examples
    hf["DbContextPoolSize"] = 2000
    hf["MetricsPort"] = 4981
    hf["ShardName"] = "main-server"
    if cluster.Spec.Hosts != nil && cluster.Spec.Hosts.CDN != "" {
        hf["CdnFullUrl"] = fmt.Sprintf("https://%s/", cluster.Spec.Hosts.CDN)
    }

    cfg["HonseFarm"] = hf

    // Kestrel
    cfg["Kestrel"] = map[string]interface{}{
        "Endpoints": map[string]interface{}{
            "Http": map[string]interface{}{
                "Url": "http://*:5000",
            },
        },
    }

    return cfg
}

func buildAdminConfig(cluster *v1alpha1.HonseFarmCluster) map[string]interface{} {
    cfg := map[string]interface{}{}

    logging := map[string]interface{}{}
    if cluster.Spec.Global != nil && cluster.Spec.Global.Logging != nil {
        lg := cluster.Spec.Global.Logging
        if lg.DefaultLevel != "" {
            logging["Default"] = lg.DefaultLevel
        }
        if lg.AspNetCoreLevel != "" {
            logging["MicrosoftAspNetCore"] = lg.AspNetCoreLevel
        }
    }
    if len(logging) > 0 {
        cfg["Logging"] = map[string]interface{}{
            "LogLevel": logging,
        }
    }

    if cluster.Spec.Global != nil && cluster.Spec.Global.Database != nil {
        db := cluster.Spec.Global.Database
        conn := fmt.Sprintf("Host=%s;Database=%s;Username=%s;Password=%s", db.Host, db.Name, db.Username, db.Password)
        cfg["ConnectionStrings"] = map[string]interface{}{
            "Database": conn,
        }
    }

    hf := map[string]interface{}{}
    if cluster.Spec.Global != nil {
        if cluster.Spec.Global.JWT != nil && cluster.Spec.Global.JWT.Secret != "" {
            hf["Jwt"] = cluster.Spec.Global.JWT.Secret
        }
        if cluster.Spec.Global.Redis != nil {
            hf["RedisConnectionString"] = cluster.Spec.Global.Redis.ConnectionString
            if cluster.Spec.Global.Redis.Pool != 0 {
                hf["RedisPool"] = cluster.Spec.Global.Redis.Pool
            }
        }
    }

    // Main server URL
    if cluster.Spec.Hosts != nil && cluster.Spec.Hosts.Server != "" {
        hf["MainServerUrl"] = fmt.Sprintf("https://%s", cluster.Spec.Hosts.Server)
    }

    hf["ConfigFilesPath"] = "/app/config"

    cfg["HonseFarm"] = hf
    cfg["AllowedHosts"] = "*"

    return cfg
}

func buildMainFileserverConfig(cluster *v1alpha1.HonseFarmCluster) map[string]interface{} {
    cfg := map[string]interface{}{}

    logging := map[string]interface{}{}
    if cluster.Spec.Global != nil && cluster.Spec.Global.Logging != nil {
        lg := cluster.Spec.Global.Logging
        if lg.DefaultLevel != "" {
            logging["Default"] = lg.DefaultLevel
        }
        if lg.MicrosoftLevel != "" {
            logging["Microsoft"] = lg.MicrosoftLevel
        }
    }
    if len(logging) > 0 {
        cfg["Logging"] = map[string]interface{}{
            "LogLevel": logging,
        }
    }

    if cluster.Spec.Global != nil && cluster.Spec.Global.Database != nil {
        db := cluster.Spec.Global.Database
        conn := fmt.Sprintf("Host=%s;Database=%s;Username=%s;Password=%s", db.Host, db.Name, db.Username, db.Password)
        cfg["ConnectionStrings"] = map[string]interface{}{
            "Database": conn,
        }
    }

    hf := map[string]interface{}{}
    if cluster.Spec.Global != nil {
        if cluster.Spec.Global.JWT != nil && cluster.Spec.Global.JWT.Secret != "" {
            hf["Jwt"] = cluster.Spec.Global.JWT.Secret
        }
        if cluster.Spec.Global.Redis != nil {
            hf["RedisConnectionString"] = cluster.Spec.Global.Redis.ConnectionString
        }
        if cluster.Spec.Global.Telemetry != nil {
            t := cluster.Spec.Global.Telemetry
            if t.LogsEndpoint != "" {
                hf["OpenTelemetryLogsEndpoint"] = t.LogsEndpoint
            }
            hf["OpenTelemetryAnalyticsOptIn"] = t.AnalyticsOptIn
            if t.AnalyticsConnectionString != "" {
                hf["OpenTelemetryAnalyticsConnectionString"] = t.AnalyticsConnectionString
            }
        }
    }

    hf["FileServerRole"] = "Main"
    hf["ServerId"] = "Forest"
    if cluster.Spec.Hosts != nil && cluster.Spec.Hosts.CDN != "" {
        hf["FileServerName"] = cluster.Spec.Hosts.CDN
        hf["ServerUri"] = fmt.Sprintf("https://%s", cluster.Spec.Hosts.CDN)
        hf["CdnFullUrl"] = fmt.Sprintf("https://%s", cluster.Spec.Hosts.CDN)
    }
    hf["CacheDirectory"] = "/cache"
    hf["CacheSizeHardLimitInGiB"] = 10
    hf["UseColdStorage"] = false
    hf["DownloadQueueSize"] = 100
    hf["DownloadQueueReleaseSeconds"] = 300
    hf["DbContextPoolSize"] = 512
    hf["MainServerAddress"] = "http://server:5000"
    hf["MetricsPort"] = 4982

    cfg["HonseFarm"] = hf

    cfg["Kestrel"] = map[string]interface{}{
        "Endpoints": map[string]interface{}{
            "Http": map[string]interface{}{
                "Url": "http://*:5001",
            },
        },
    }

    return cfg
}

func buildShardFileserverConfig(cluster *v1alpha1.HonseFarmCluster, shard *v1alpha1.ShardSpec) map[string]interface{} {
    cfg := map[string]interface{}{}

    logging := map[string]interface{}{}
    if cluster.Spec.Global != nil && cluster.Spec.Global.Logging != nil {
        lg := cluster.Spec.Global.Logging
        if lg.DefaultLevel != "" {
            logging["Default"] = lg.DefaultLevel
        }
        if lg.MicrosoftLevel != "" {
            logging["Microsoft"] = lg.MicrosoftLevel
        }
    }
    if len(logging) > 0 {
        cfg["Logging"] = map[string]interface{}{
            "LogLevel": logging,
        }
    }

    if cluster.Spec.Global != nil && cluster.Spec.Global.Database != nil {
        db := cluster.Spec.Global.Database
        conn := fmt.Sprintf("Host=%s;Database=%s;Username=%s;Password=%s", db.Host, db.Name, db.Username, db.Password)
        cfg["ConnectionStrings"] = map[string]interface{}{
            "Database": conn,
        }
    }

    hf := map[string]interface{}{}
    if cluster.Spec.Global != nil {
        if cluster.Spec.Global.JWT != nil && cluster.Spec.Global.JWT.Secret != "" {
            hf["Jwt"] = cluster.Spec.Global.JWT.Secret
        }
        if cluster.Spec.Global.Redis != nil {
            hf["RedisConnectionString"] = cluster.Spec.Global.Redis.ConnectionString
        }
        if cluster.Spec.Global.Telemetry != nil {
            t := cluster.Spec.Global.Telemetry
            if t.LogsEndpoint != "" {
                hf["OpenTelemetryLogsEndpoint"] = t.LogsEndpoint
            }
            hf["OpenTelemetryAnalyticsOptIn"] = t.AnalyticsOptIn
            if t.AnalyticsConnectionString != "" {
                hf["OpenTelemetryAnalyticsConnectionString"] = t.AnalyticsConnectionString
            }
        }
    }

    hf["FileServerRole"] = "Shard"
    hf["ServerId"] = "Forest"
    shardHost := shard.Name
    if cluster.Spec.Hosts != nil {
        for _, hs := range cluster.Spec.Hosts.Shards {
            if hs.Name == shard.Name && hs.Host != "" {
                shardHost = hs.Host
                break
            }
        }
    }
    hf["FileServerName"] = shardHost
    hf["ServerUri"] = fmt.Sprintf("https://%s", shardHost)
    hf["CacheDirectory"] = "/cache"
    hf["CacheSizeHardLimitInGiB"] = 100
    hf["UseColdStorage"] = false
    hf["ColdStorageDirectory"] = nil
    hf["ColdStorageSizeHardLimitInGiB"] = 0
    hf["ColdStorageUnusedFileRetentionPeriodInDays"] = 90
    hf["UnusedFileRetentionPeriodInDays"] = 7
    hf["DownloadQueueSize"] = 100
    hf["DownloadQueueReleaseSeconds"] = 300
    hf["DbContextPoolSize"] = 512
    hf["MainServerAddress"] = "http://server:5000"
    hf["MainFileServerAddress"] = "http://main-fileserver:5001"
    hf["DistributionFileServerAddress"] = "http://main-fileserver:5001"
    hf["MetricsPort"] = 4983

    // Simple shard configuration stub; can be overridden via configOverrides.
    hf["ShardConfiguration"] = map[string]interface{}{
        "Continents": []string{"*"},
        "FileMatch":  "^[0-9a-fA-F]",
        "RegionUris": map[string]interface{}{
            "Default": fmt.Sprintf("https://%s", shardHost),
        },
    }

    cfg["HonseFarm"] = hf

    cfg["Kestrel"] = map[string]interface{}{
        "Endpoints": map[string]interface{}{
            "Http": map[string]interface{}{
                "Url": "http://*:5002",
            },
        },
    }

    return cfg
}
