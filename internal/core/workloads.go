package core

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "honsefarm-operator/api/v1alpha1"
)

func namespaceFor(cluster *v1alpha1.HonseFarmCluster) string {
	if cluster.Spec.Namespace != "" {
		return cluster.Spec.Namespace
	}
	return "honsefarm"
}

// EnsureServerWorkload creates/updates PVC + Deployment for the core server.
func EnsureServerWorkload(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	cluster *v1alpha1.HonseFarmCluster,
) error {
	if cluster.Spec.Components == nil || cluster.Spec.Components.Server == nil {
		// server disabled
		return nil
	}
	if cluster.Spec.Images == nil || cluster.Spec.Images.Server == "" {
		return fmt.Errorf("spec.images.server must be set")
	}

	ns := namespaceFor(cluster)
	comp := cluster.Spec.Components.Server

	// PVC (optional)
	var pvc *corev1.PersistentVolumeClaim
	var err error
	if comp.Storage != nil && comp.Storage.Size != "" {
		pvc, err = ensurePVC(ctx, c, scheme, cluster, "server-data", comp.Storage)
		if err != nil {
			return fmt.Errorf("ensure server pvc: %w", err)
		}
	}

	// Deployment
	replicas := int32(1)
	if comp.Replicas != nil {
		replicas = *comp.Replicas
	}

	return ensureDeployment(ctx, c, scheme, cluster, &DeploymentSpec{
		Name:            "honsefarm-server",
		Namespace:       ns,
		Component:       "server",
		Image:           cluster.Spec.Images.Server,
		Replicas:        replicas,
		ContainerPort:   5000,
		ConfigMountPath: "/app/config",
		PVC:             pvc,
		Env:             nil, // can be extended later if needed
	})
}

// EnsureAdminWorkload creates/updates PVC + Deployment for the admin panel.
func EnsureAdminWorkload(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	cluster *v1alpha1.HonseFarmCluster,
) error {
	if cluster.Spec.Components == nil || cluster.Spec.Components.AdminPanel == nil {
		// admin panel disabled
		return nil
	}
	if cluster.Spec.Images == nil || cluster.Spec.Images.AdminPanel == "" {
		return fmt.Errorf("spec.images.adminPanel must be set")
	}

	ns := namespaceFor(cluster)
	comp := cluster.Spec.Components.AdminPanel

	var pvc *corev1.PersistentVolumeClaim
	var err error
	if comp.Storage != nil && comp.Storage.Size != "" {
		pvc, err = ensurePVC(ctx, c, scheme, cluster, "adminpanel-data", comp.Storage)
		if err != nil {
			return fmt.Errorf("ensure adminpanel pvc: %w", err)
		}
	}

	replicas := int32(1)
	if comp.Replicas != nil {
		replicas = *comp.Replicas
	}

	return ensureDeployment(ctx, c, scheme, cluster, &DeploymentSpec{
		Name:            "honsefarm-adminpanel",
		Namespace:       ns,
		Component:       "adminpanel",
		Image:           cluster.Spec.Images.AdminPanel,
		Replicas:        replicas,
		ContainerPort:   5000,
		ConfigMountPath: "/app/config",
		PVC:             pvc,
		Env:             nil,
	})
}

// EnsureMainFileserverWorkload creates/updates PVC + Deployment for the main fileserver.
func EnsureMainFileserverWorkload(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	cluster *v1alpha1.HonseFarmCluster,
) error {
	if cluster.Spec.Components == nil ||
		cluster.Spec.Components.Fileservers == nil ||
		cluster.Spec.Components.Fileservers.Main == nil {
		// main fileserver disabled
		return nil
	}
	if cluster.Spec.Images == nil || cluster.Spec.Images.MainFileserver == "" {
		return fmt.Errorf("spec.images.mainFileserver must be set")
	}

	ns := namespaceFor(cluster)
	comp := cluster.Spec.Components.Fileservers.Main

	var pvc *corev1.PersistentVolumeClaim
	var err error
	if comp.Storage != nil && comp.Storage.Size != "" {
		pvc, err = ensurePVC(ctx, c, scheme, cluster, "main-fileserver-data", comp.Storage)
		if err != nil {
			return fmt.Errorf("ensure main-fileserver pvc: %w", err)
		}
	}

	replicas := int32(1)
	if comp.Replicas != nil {
		replicas = *comp.Replicas
	}

	return ensureDeployment(ctx, c, scheme, cluster, &DeploymentSpec{
		Name:            "honsefarm-main-fileserver",
		Namespace:       ns,
		Component:       "main-fileserver",
		Image:           cluster.Spec.Images.MainFileserver,
		Replicas:        replicas,
		ContainerPort:   5001,
		ConfigMountPath: "/app/config",
		PVC:             pvc,
		Env:             nil,
	})
}

// EnsureShardWorkloads creates/updates PVCs + Deployments for all configured shards.
func EnsureShardWorkloads(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	cluster *v1alpha1.HonseFarmCluster,
) error {
	if cluster.Spec.Components == nil ||
		cluster.Spec.Components.Fileservers == nil {
		return nil
	}
	if cluster.Spec.Images == nil || cluster.Spec.Images.ShardFileserver == "" {
		return fmt.Errorf("spec.images.shardFileserver must be set")
	}

	ns := namespaceFor(cluster)

	for _, shard := range cluster.Spec.Components.Fileservers.Shards {
		// Each shard gets its own PVC + Deployment
		var pvc *corev1.PersistentVolumeClaim
		var err error
		if shard.Storage != nil && shard.Storage.Size != "" {
			pvcName := fmt.Sprintf("shard-%s-data", shard.Name)
			pvc, err = ensurePVC(ctx, c, scheme, cluster, pvcName, shard.Storage)
			if err != nil {
				return fmt.Errorf("ensure shard pvc %s: %w", shard.Name, err)
			}
		}

		replicas := int32(1)
		if shard.Replicas != nil {
			replicas = *shard.Replicas
		}

		depName := fmt.Sprintf("honsefarm-shard-%s", shard.Name)

		if err := ensureDeployment(ctx, c, scheme, cluster, &DeploymentSpec{
			Name:            depName,
			Namespace:       ns,
			Component:       "shard-fileserver",
			ShardName:       shard.Name,
			Image:           cluster.Spec.Images.ShardFileserver,
			Replicas:        replicas,
			ContainerPort:   5002,
			ConfigMountPath: "/app/config",
			PVC:             pvc,
			Env:             nil,
		}); err != nil {
			return fmt.Errorf("ensure shard deployment %s: %w", shard.Name, err)
		}
	}

	return nil
}

// ---- helpers ----

type DeploymentSpec struct {
	Name            string
	Namespace       string
	Component       string
	ShardName       string
	Image           string
	Replicas        int32
	ContainerPort   int32
	ConfigMountPath string
	PVC             *corev1.PersistentVolumeClaim
	Env             []corev1.EnvVar
}

func ensurePVC(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	cluster *v1alpha1.HonseFarmCluster,
	name string,
	storage *v1alpha1.StorageSpec,
) (*corev1.PersistentVolumeClaim, error) {
	ns := namespaceFor(cluster)

	var existing corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, &existing); err == nil {
		// PVC already exists; don't try to mutate size/class here to avoid conflicts
		return &existing, nil
	} else if !errors.IsNotFound(err) {
		return nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "honsefarm-operator",
				"honsefarm-pvc":                name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storage.Size),
				},
			},
		},
	}

	// AccessModes
	if len(storage.AccessModes) > 0 {
		for _, m := range storage.AccessModes {
			pvc.Spec.AccessModes = append(pvc.Spec.AccessModes, corev1.PersistentVolumeAccessMode(m))
		}
	} else {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	// StorageClass
	if storage.StorageClassName != "" {
		pvc.Spec.StorageClassName = &storage.StorageClassName
	}

	if err := ctrl.SetControllerReference(cluster, pvc, scheme); err != nil {
		return nil, err
	}
	if err := c.Create(ctx, pvc); err != nil {
		return nil, err
	}
	return pvc, nil
}

func ensureDeployment(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	cluster *v1alpha1.HonseFarmCluster,
	spec *DeploymentSpec,
) error {
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "honsefarm-operator",
		"honsefarm-component":          spec.Component,
	}
	if spec.ShardName != "" {
		labels["honsefarm-shard"] = spec.ShardName
	}

	var existing appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: spec.Name, Namespace: spec.Namespace}, &existing); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// Create new Deployment
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      spec.Name,
				Namespace: spec.Namespace,
				Labels:    labels,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &spec.Replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  spec.Component,
								Image: spec.Image,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: spec.ContainerPort,
									},
								},
								Env: spec.Env,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "config",
										MountPath: spec.ConfigMountPath,
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
											Name: "honsefarm-config",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Attach PVC if present
		if spec.PVC != nil {
			dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: spec.PVC.Name,
					},
				},
			})
			dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(
				dep.Spec.Template.Spec.Containers[0].VolumeMounts,
				corev1.VolumeMount{
					Name:      "data",
					MountPath: "/data",
				},
			)
		}

		if err := ctrl.SetControllerReference(cluster, dep, scheme); err != nil {
			return err
		}
		return c.Create(ctx, dep)
	}

	// Update path
	existing.Spec.Replicas = &spec.Replicas
	if len(existing.Spec.Template.Spec.Containers) > 0 {
		existing.Spec.Template.Spec.Containers[0].Image = spec.Image
	}
	return c.Update(ctx, &existing)
}
