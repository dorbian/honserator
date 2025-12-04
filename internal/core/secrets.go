package core

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/types"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"

    "honsefarm-operator/internal/certs"
    v1alpha1 "honsefarm-operator/api/v1alpha1"
)

const (
    CoreSecretName = "honsefarm-secrets"
    TLSSecretName  = "honsefarm-tls"
)

// EnsureCoreSecrets makes sure the main secret containing random keys exists,
// and that a TLS secret with a self-signed certificate is present.
func EnsureCoreSecrets(ctx context.Context, c client.Client, cluster *v1alpha1.HonseFarmCluster) error {
    ns := cluster.Spec.Namespace
    if ns == "" {
        ns = cluster.Namespace
    }

    // Main secret with random keys
    var sec corev1.Secret
    err := c.Get(ctx, types.NamespacedName{Name: CoreSecretName, Namespace: ns}, &sec)
    if err != nil {
        if !errors.IsNotFound(err) {
            return err
        }
        data := map[string][]byte{
            "federationKey": randomBytes(32),
            "jwtSigningKey": randomBytes(32),
            "adminPassword": []byte(randomString(16)),
        }
        sec = corev1.Secret{
            ObjectMeta: metav1Object(cluster, CoreSecretName, ns),
            Data:       data,
            Type:       corev1.SecretTypeOpaque,
        }
        if err := c.Create(ctx, &sec); err != nil {
            return err
        }
    }

    // TLS secret for internal services
    var tlsSec corev1.Secret
    err = c.Get(ctx, types.NamespacedName{Name: TLSSecretName, Namespace: ns}, &tlsSec)
    if err != nil {
        if !errors.IsNotFound(err) {
            return err
        }

        // Collect DNS names from apiDomain and cloudflared ingress
        dnsNames := []string{}
        if cluster.Spec.APIDomain != "" {
            dnsNames = append(dnsNames, cluster.Spec.APIDomain)
        }
        if cluster.Spec.Cloudflared != nil {
            for _, ing := range cluster.Spec.Cloudflared.Ingress {
                if ing.Hostname != "" {
                    dnsNames = append(dnsNames, ing.Hostname)
                }
            }
        }

        certPEM, keyPEM, err := certs.GenerateSelfSignedCert(cluster.Spec.APIDomain, dnsNames)
        if err != nil {
            return fmt.Errorf("failed to generate self-signed cert: %w", err)
        }

        tlsSec = corev1.Secret{
            ObjectMeta: metav1Object(cluster, TLSSecretName, ns),
            Type:       corev1.SecretTypeTLS,
            Data: map[string][]byte{
                corev1.TLSCertKey:       certPEM,
                corev1.TLSPrivateKeyKey: keyPEM,
            },
        }
        if err := c.Create(ctx, &tlsSec); err != nil {
            return err
        }
    }

    return nil
}

func randomBytes(n int) []byte {
    b := make([]byte, n)
    _, _ = rand.Read(b)
    return []byte(base64.RawURLEncoding.EncodeToString(b))
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
    b := make([]byte, n)
    _, _ = rand.Read(b)
    for i := range b {
        b[i] = letters[int(b[i])%len(letters)]
    }
    return string(b)
}

func metav1Object(cluster *v1alpha1.HonseFarmCluster, name, ns string) metav1.ObjectMeta {
    return metav1.ObjectMeta{
        Name:      name,
        Namespace: ns,
        Labels: map[string]string{
            "app.kubernetes.io/managed-by": "honsefarm-operator",
            "honsefarm.clusters.honse.farm/cluster": cluster.Name,
        },
    }
}
