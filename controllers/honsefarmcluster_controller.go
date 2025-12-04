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
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "honsefarm-operator/api/v1alpha1"
	cfginternal "honsefarm-operator/internal/config"
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
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets;configmaps;services;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// workloads
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

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

	// Ensure core workloads (PVCs + Deployments)
	if err := coreinternal.EnsureServerWorkload(ctx, r.Client, r.Scheme, &cluster); err != nil {
		logger.Error(err, "failed to ensure server workload")
		return ctrl.Result{}, err
	}
	if err := coreinternal.EnsureAdminWorkload(ctx, r.Client, r.Scheme, &cluster); err != nil {
		logger.Error(err, "failed to ensure admin workload")
		return ctrl.Result{}, err
	}
	if err := coreinternal.EnsureMainFileserverWorkload(ctx, r.Client, r.Scheme, &cluster); err != nil {
		logger.Error(err, "failed to ensure main fileserver workload")
		return ctrl.Result{}, err
	}
	if err := coreinternal.EnsureShardWorkloads(ctx, r.Client, r.Scheme, &cluster); err != nil {
		logger.Error(err, "failed to ensure shard workloads")
		return ctrl.Result{}, err
	}

	// Ensure Services for core components and shards
	if err := r.ensureCoreServices(ctx, &cluster); err != nil {
		logger.Error(err, "failed to ensure core services")
		return ctrl.Result{}, err
	}
	if err := r.ensureShardServices(ctx, &cluster); err != nil {
		logger.Error(err, "failed to ensure shard services")
		return ctrl.Result{}, err
	}

	// Set phase Ready for now
	cluster.Status.Phase = "Ready"
	if err := r.Status().Update(ctx, &cluster); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *HonseFarmClusterReconciler) ensureCoreServices(ctx context.Context, cluster *v1alpha1.HonseFarmCluster) error {
	ns := cluster.Spec.Namespace
	if ns == "" {
		ns = "honsefarm"
	}

	// server-svc: targets honsefarm-component=server on port 5000
	if err := r.ensureService(ctx, cluster, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "server-svc",
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "honsefarm-operator",
				"app.kubernetes.io/name":       "server-svc",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"honsefarm-component": "server",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       5000,
					TargetPort: intstr.FromInt(5000),
				},
			},
		},
	}); err != nil {
		return err
	}

	// adminpanel-svc: targets honsefarm-component=adminpanel on port 5000
	if err := r.ensureService(ctx, cluster, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "adminpanel-svc",
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "honsefarm-operator",
				"app.kubernetes.io/name":       "adminpanel-svc",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"honsefarm-component": "adminpanel",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       5000,
					TargetPort: intstr.FromInt(5000),
				},
			},
		},
	}); err != nil {
		return err
	}

	// main-fileserver-svc: targets honsefarm-component=main-fileserver on port 5001
	if err := r.ensureService(ctx, cluster, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main-fileserver-svc",
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "honsefarm-operator",
				"app.kubernetes.io/name":       "main-fileserver-svc",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"honsefarm-component": "main-fileserver",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       5001,
					TargetPort: intstr.FromInt(5001),
				},
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func (r *HonseFarmClusterReconciler) ensureShardServices(ctx context.Context, cluster *v1alpha1.HonseFarmCluster) error {
	if cluster.Spec.Components == nil ||
		cluster.Spec.Components.Fileservers == nil {
		return nil
	}

	ns := cluster.Spec.Namespace
	if ns == "" {
		ns = "honsefarm"
	}

	for _, shard := range cluster.Spec.Components.Fileservers.Shards {
		svcName := fmt.Sprintf("shard-%s-svc", shard.Name)
		labels := map[string]string{
			"app.kubernetes.io/managed-by": "honsefarm-operator",
			"app.kubernetes.io/name":       svcName,
		}

		if err := r.ensureService(ctx, cluster, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: ns,
				Labels:    labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"honsefarm-component": "shard-fileserver",
					"honsefarm-shard":     shard.Name,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Port:       5002,
						TargetPort: intstr.FromInt(5002),
					},
				},
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *HonseFarmClusterReconciler) ensureService(ctx context.Context, cluster *v1alpha1.HonseFarmCluster, svc *corev1.Service) error {
	if err := ctrl.SetControllerReference(cluster, svc, r.Scheme); err != nil {
		return err
	}

	var existing corev1.Service
	if err := r.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, &existing); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, svc)
		}
		return err
	}

	existing.Spec.Ports = svc.Spec.Ports
	existing.Spec.Selector = svc.Spec.Selector
	return r.Update(ctx, &existing)
}

func (r *HonseFarmClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.HonseFarmCluster{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
