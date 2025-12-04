package controllers

import (
    "context"
    "time"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"

    v1alpha1 "honsefarm-operator/api/v1alpha1"
    coreinternal "honsefarm-operator/internal/core"
    cfginternal "honsefarm-operator/internal/config"
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
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets;configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *HonseFarmClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    var cluster v1alpha1.HonseFarmCluster
    if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
        if errors.IsNotFound(err) {
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
                logger.Error(err, "failed to create namespace", "namespace", targetNS)
                return ctrl.Result{}, err
            }
            logger.Info("created namespace for HonseFarmCluster", "namespace", targetNS)
        } else {
            return ctrl.Result{}, err
        }
    }

    // Ensure core secret exists
    if err := coreinternal.EnsureCoreSecret(ctx, r.Client, &cluster); err != nil {
        logger.Error(err, "failed to ensure core secret")
        return ctrl.Result{}, err
    }

    // Ensure config ConfigMap exists / is updated
    cm, err := cfginternal.BuildConfigMap(&cluster)
    if err != nil {
        logger.Error(err, "failed to build config ConfigMap")
        return ctrl.Result{}, err
    }
    if err := ctrl.SetControllerReference(&cluster, cm, r.Scheme); err != nil {
        logger.Error(err, "failed to set owner reference on config ConfigMap")
        return ctrl.Result{}, err
    }

    var existingCM corev1.ConfigMap
    err = r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, &existingCM)
    if err != nil {
        if errors.IsNotFound(err) {
            if err := r.Create(ctx, cm); err != nil {
                logger.Error(err, "failed to create config ConfigMap")
                return ctrl.Result{}, err
            }
            logger.Info("created honsefarm config ConfigMap", "namespace", cm.Namespace, "name", cm.Name)
        } else {
            return ctrl.Result{}, err
        }
    } else {
        existingCM.Data = cm.Data
        if err := r.Update(ctx, &existingCM); err != nil {
            logger.Error(err, "failed to update config ConfigMap")
            return ctrl.Result{}, err
        }
    }

    // Set phase Ready for now
    cluster.Status.Phase = "Ready"
    if err := r.Status().Update(ctx, &cluster); err != nil {
        logger.Error(err, "failed to update status")
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }

    return ctrl.Result{}, nil
}

func (r *HonseFarmClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1alpha1.HonseFarmCluster{}).
        Owns(&corev1.ConfigMap{}).
        Owns(&corev1.Secret{}).
        Complete(r)
}
