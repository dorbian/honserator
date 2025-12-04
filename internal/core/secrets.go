package core

import (
    "context"
    "crypto/rand"
    "encoding/base64"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/types"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"

    v1alpha1 "honsefarm-operator/api/v1alpha1"
)

const (
    CoreSecretName = "honsefarm-secrets"
)

// EnsureCoreSecret ensures a core secret with random credentials exists.
// It does NOT currently propagate back into spec.global; the operator
// uses spec.global values as source of truth for config generation.
func EnsureCoreSecret(ctx context.Context, c client.Client, cluster *v1alpha1.HonseFarmCluster) error {
    ns := cluster.Spec.Namespace
    if ns == "" {
        ns = "honsefarm"
    }

    var sec corev1.Secret
    err := c.Get(ctx, types.NamespacedName{Name: CoreSecretName, Namespace: ns}, &sec)
    if err == nil {
        return nil
    }
    if !errors.IsNotFound(err) {
        return err
    }

    data := map[string][]byte{
        "jwtSecret":       randomBytes(32),
        "databasePassword": randomBytes(24),
        "redisPassword":   randomBytes(24),
    }

    sec = corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      CoreSecretName,
            Namespace: ns,
            Labels: map[string]string{
                "app.kubernetes.io/managed-by": "honsefarm-operator",
            },
        },
        Type: corev1.SecretTypeOpaque,
        Data: data,
    }

    return c.Create(ctx, &sec)
}

func randomBytes(n int) []byte {
    b := make([]byte, n)
    _, _ = rand.Read(b)
    return []byte(base64.RawURLEncoding.EncodeToString(b))
}
