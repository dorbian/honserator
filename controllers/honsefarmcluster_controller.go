package controllers

import (
    "context"
    "fmt"
    "time"

    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"

    v1alpha1 "honsefarm-operator/api/v1alpha1"
    "honsefarm-operator/internal/cloudflared"
    coreinternal "honsefarm-operator/internal/core"
)

// HonseFarmClusterReconciler reconciles a HonseFarmCluster object
type HonseFarmClusterReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=clusters.honse.farm,resources=honsefarmclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=clusters.honse.farm,resources=honsefarmclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=clusters.honse.farm,resources=honsefarmclusters/finalizers,verbs=update

// core objects
//+kubebuilder:rbac:groups="",resources=namespaces;secrets;configmaps;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete

func (r *HonseFarmClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    var cluster v1alpha1.HonseFarmCluster
    if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
        if errors.IsNotFound(err) {
            // CR deleted, nothing to do
            return ctrl.Result{}, nil
        }
        return ctrl.Result{}, err
    }

    // Ensure target namespace exists
    targetNS := cluster.Spec.Namespace
    if targetNS == "" {
        targetNS = "honsefarm"
    }
    var ns corev1.Namespace
    if err := r.Get(ctx, types.NamespacedName{Name: targetNS}, &ns); err != nil {
        if errors.IsNotFound(err) {
            ns = corev1.Namespace{
                ObjectMeta: metav1.ObjectMeta{
                    Name: targetNS,
                    Labels: map[string]string{
                        "app.kubernetes.io/managed-by": "honsefarm-operator",
                    },
                },
            }
            if err := r.Create(ctx, &ns); err != nil {
                return ctrl.Result{}, err
            }
        } else {
            return ctrl.Result{}, err
        }
    }

    // Ensure core secrets & TLS
    if err := coreinternal.EnsureCoreSecrets(ctx, r.Client, &cluster); err != nil {
        logger.Error(err, "failed to ensure core secrets")
        return ctrl.Result{}, err
    }

    // TODO: deploy core HonseFarm components (server, fileservers, adminpanel)

    // Cloudflared management
    if cluster.Spec.Cloudflared != nil && cluster.Spec.Cloudflared.Enabled {
        if err := r.reconcileCloudflared(ctx, &cluster); err != nil {
            logger.Error(err, "failed to reconcile cloudflared")
            r.setCloudflaredStatus(&cluster, false, err.Error())
            _ = r.Status().Update(ctx, &cluster)
            return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
        }
        r.setCloudflaredStatus(&cluster, true, "")
    } else {
        r.setCloudflaredStatus(&cluster, false, "cloudflared disabled")
    }

    // Update status
    cluster.Status.Phase = "Ready"
    if err := r.Status().Update(ctx, &cluster); err != nil {
        logger.Error(err, "failed to update status")
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}

func (r *HonseFarmClusterReconciler) reconcileCloudflared(ctx context.Context, cluster *v1alpha1.HonseFarmCluster) error {
    // ConfigMap
    cfg := cloudflared.BuildConfigMap(cluster)
    if cfg != nil {
        if err := ctrl.SetControllerReference(cluster, cfg, r.Scheme); err != nil {
            return fmt.Errorf("set owner on cloudflared config: %w", err)
        }

        var existing corev1.ConfigMap
        err := r.Get(ctx, types.NamespacedName{Name: cfg.Name, Namespace: cfg.Namespace}, &existing)
        if err != nil {
            if errors.IsNotFound(err) {
                if err := r.Create(ctx, cfg); err != nil {
                    return fmt.Errorf("create cloudflared configmap: %w", err)
                }
            } else {
                return err
            }
        } else {
            existing.Data = cfg.Data
            if err := r.Update(ctx, &existing); err != nil {
                return fmt.Errorf("update cloudflared configmap: %w", err)
            }
        }
    }

    // Deployment
    dep := cloudflared.BuildDeployment(cluster)
    if dep != nil {
        if err := ctrl.SetControllerReference(cluster, dep, r.Scheme); err != nil {
            return fmt.Errorf("set owner on cloudflared deployment: %w", err)
        }

        var existing appsv1.Deployment
        err := r.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, &existing)
        if err != nil {
            if errors.IsNotFound(err) {
                if err := r.Create(ctx, dep); err != nil {
                    return fmt.Errorf("create cloudflared deployment: %w", err)
                }
            } else {
                return err
            }
        } else {
            existing.Spec = dep.Spec
            if err := r.Update(ctx, &existing); err != nil {
                return fmt.Errorf("update cloudflared deployment: %w", err)
            }
        }
    }

    return nil
}

func (r *HonseFarmClusterReconciler) setCloudflaredStatus(cluster *v1alpha1.HonseFarmCluster, ready bool, msg string) {
    if cluster.Status.CloudflaredStatus == nil {
        cluster.Status.CloudflaredStatus = &v1alpha1.CloudflaredStatus{}
    }
    cluster.Status.CloudflaredStatus.Ready = ready
    cluster.Status.CloudflaredStatus.LastError = msg
}

// SetupWithManager sets up the controller with the Manager.
func (r *HonseFarmClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1alpha1.HonseFarmCluster{}).
        Owns(&appsv1.Deployment{}).
        Owns(&corev1.ConfigMap{}).
        Owns(&corev1.Secret{}).
        Complete(r)
}
