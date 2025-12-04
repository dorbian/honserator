package cloudflared

import (
    "fmt"
    "strings"

    corev1 "k8s.io/api/core/v1"
    appsv1 "k8s.io/api/apps/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    v1alpha1 "honsefarm-operator/api/v1alpha1"
)

const (
    ConfigMapName  = "cloudflared-config"
    DeploymentName = "cloudflared"
)

// BuildConfigMap builds a cloudflared config ConfigMap from the CR spec.
func BuildConfigMap(cluster *v1alpha1.HonseFarmCluster) *corev1.ConfigMap {
    cf := cluster.Spec.Cloudflared
    if cf == nil {
        return nil
    }

    lines := []string{
        fmt.Sprintf("tunnel: %s", cf.TunnelID),
        "credentials-file: /etc/cloudflared/creds/credentials.json",
        "ingress:",
    }

    for _, rule := range cf.Ingress {
        if rule.SpecialService != "" {
            lines = append(lines, fmt.Sprintf("  - service: %s", rule.SpecialService))
            continue
        }
        if rule.Hostname == "" || rule.ServiceName == "" || rule.ServiceNamespace == "" || rule.ServicePort == 0 {
            continue
        }
        backend := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", rule.ServiceName, rule.ServiceNamespace, rule.ServicePort)
        lines = append(lines, fmt.Sprintf("  - hostname: %s", rule.Hostname))
        lines = append(lines, fmt.Sprintf("    service: %s", backend))
    }

    yaml := strings.Join(lines, "\n")

    return &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      ConfigMapName,
            Namespace: cluster.Spec.Namespace,
            Labels: map[string]string{
                "app": "cloudflared",
            },
        },
        Data: map[string]string{
            "config.yaml": yaml,
        },
    }
}

// BuildDeployment builds a cloudflared Deployment from the CR spec.
func BuildDeployment(cluster *v1alpha1.HonseFarmCluster) *appsv1.Deployment {
    cf := cluster.Spec.Cloudflared
    if cf == nil {
        return nil
    }

    image := cf.Image
    if image == "" {
        image = "ghcr.io/cloudflare/cloudflared:latest"
    }

    args := []string{
        "tunnel",
        "run",
    }
    if len(cf.ExtraArgs) > 0 {
        args = append(args, cf.ExtraArgs...)
    }

    credSecretName := ""
    if cf.CredentialsSecretRef != nil && cf.CredentialsSecretRef.Name != "" {
        credSecretName = cf.CredentialsSecretRef.Name
    } else {
        credSecretName = "cloudflared-credentials"
    }

    ns := cluster.Spec.Namespace
    if ns == "" {
        ns = cluster.Namespace
    }

    deploy := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      DeploymentName,
            Namespace: ns,
            Labels: map[string]string{
                "app": "cloudflared",
            },
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: int32Ptr(1),
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": "cloudflared"},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": "cloudflared"},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "cloudflared",
                            Image: image,
                            Args:  args,
                            VolumeMounts: []corev1.VolumeMount{
                                {
                                    Name:      "config",
                                    MountPath: "/etc/cloudflared",
                                },
                                {
                                    Name:      "credentials",
                                    MountPath: "/etc/cloudflared/creds",
                                    ReadOnly:  true,
                                },
                            },
                        },
                    },
                    Volumes: []corev1.Volume{
                        {
                            Name: "config",
                            VolumeSource: corev1.VolumeSource{
                                ConfigMap: &corev1.ConfigMapVolumeSource{
                                    LocalObjectReference: corev1.LocalObjectReference{
                                        Name: ConfigMapName,
                                    },
                                },
                            },
                        },
                        {
                            Name: "credentials",
                            VolumeSource: corev1.VolumeSource{
                                Secret: &corev1.SecretVolumeSource{
                                    SecretName: credSecretName,
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    return deploy
}

func int32Ptr(i int32) *int32 { return &i }
