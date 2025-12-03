package controllers

import (
    "context"
    "fmt"

    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"

    honsev1alpha1 "git.honse.farm/astraea/honse-operator/api/v1alpha1"
)

type HonseClusterReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *HonseClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx).WithValues("honsecluster", req.NamespacedName)

    var hc honsev1alpha1.HonseCluster
    if err := r.Get(ctx, req.NamespacedName, &hc); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    mode := hc.Spec.DeploymentMode
    if mode == "" {
        mode = "Prebuilt"
    }

    if mode != "Prebuilt" {
        logger.Info("DeploymentMode is not Prebuilt; nothing to do", "mode", mode)
        return ctrl.Result{}, nil
    }

    if hc.Spec.Images.Server != "" {
        if err := r.ensureDeployment(ctx, &hc, "honse-server", hc.Spec.Images.Server, 5000); err != nil {
            logger.Error(err, "failed to ensure server deployment")
            return ctrl.Result{}, err
        }
    }

    if hc.Spec.Images.MainFileserver != "" {
        if err := r.ensureDeployment(ctx, &hc, "honse-main-fileserver", hc.Spec.Images.MainFileserver, 5001); err != nil {
            logger.Error(err, "failed to ensure main fileserver deployment")
            return ctrl.Result{}, err
        }
    }

    if hc.Spec.Images.ShardFileserver != "" {
        if err := r.ensureDeployment(ctx, &hc, "honse-shard-fileserver", hc.Spec.Images.ShardFileserver, 5002); err != nil {
            logger.Error(err, "failed to ensure shard fileserver deployment")
            return ctrl.Result{}, err
        }
    }

    if hc.Spec.Images.Adminpanel != "" {
        if err := r.ensureDeployment(ctx, &hc, "honse-adminpanel", hc.Spec.Images.Adminpanel, 8080); err != nil {
            logger.Error(err, "failed to ensure adminpanel deployment")
            return ctrl.Result{}, err
        }
    }

    hc.Status.Phase = "Ready"
    if err := r.Status().Update(ctx, &hc); err != nil {
        logger.Error(err, "failed to update status")
        return ctrl.Result{}, err
    }

    logger.Info("Reconcile completed")
    return ctrl.Result{}, nil
}

func (r *HonseClusterReconciler) ensureDeployment(ctx context.Context, hc *honsev1alpha1.HonseCluster, name, image string, port int32) error {
    depName := fmt.Sprintf("%s-%s", hc.Name, name)
    var dep appsv1.Deployment
    key := types.NamespacedName{Name: depName, Namespace: hc.Namespace}

    replicas := int32(1)

    err := r.Get(ctx, key, &dep)
    if err != nil {
        if client.IgnoreNotFound(err) != nil {
            return err
        }
        dep = appsv1.Deployment{
            ObjectMeta: metav1.ObjectMeta{
                Name:      depName,
                Namespace: hc.Namespace,
                Labels: map[string]string{
                    "app":          name,
                    "honsecluster": hc.Name,
                },
            },
            Spec: appsv1.DeploymentSpec{
                Replicas: &replicas,
                Selector: &metav1.LabelSelector{
                    MatchLabels: map[string]string{
                        "app":          name,
                        "honsecluster": hc.Name,
                    },
                },
                Template: corev1.PodTemplateSpec{
                    ObjectMeta: metav1.ObjectMeta{
                        Labels: map[string]string{
                            "app":          name,
                            "honsecluster": hc.Name,
                        },
                    },
                    Spec: corev1.PodSpec{
                        Containers: []corev1.Container{
                            {
                                Name:  name,
                                Image: image,
                                Ports: []corev1.ContainerPort{
                                    {ContainerPort: port},
                                },
                            },
                        },
                    },
                },
            },
        }
        if err := ctrl.SetControllerReference(hc, &dep, r.Scheme); err != nil {
            return err
        }
        return r.Create(ctx, &dep)
    }

    updated := false
    if len(dep.Spec.Template.Spec.Containers) == 0 {
        dep.Spec.Template.Spec.Containers = []corev1.Container{{
            Name:  name,
            Image: image,
            Ports: []corev1.ContainerPort{{ContainerPort: port}},
        }}
        updated = true
    } else if dep.Spec.Template.Spec.Containers[0].Image != image {
        dep.Spec.Template.Spec.Containers[0].Image = image
        updated = true
    }

    if updated {
        return r.Update(ctx, &dep)
    }

    return nil
}

func (r *HonseClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&honsev1alpha1.HonseCluster{}).
        Owns(&appsv1.Deployment{}).
        Complete(r)
}
