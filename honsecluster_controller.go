package controllers

import (
    "context"
    "fmt"
    "strings"
    "time"

    honsev1alpha1 "git.honse.farm/astraea/honse-operator/api/v1alpha1"

    appsv1 "k8s.io/api/apps/v1"
    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/apimachinery/pkg/util/intstr"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
    buildJobNamePrefix = "honse-build-"
)

type HonseClusterReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *HonseClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    var hc honsev1alpha1.HonseCluster
    if err := r.Get(ctx, req.NamespacedName, &hc); err != nil {
        if errors.IsNotFound(err) {
            return ctrl.Result{}, nil
        }
        return ctrl.Result{}, err
    }

    if !hc.ObjectMeta.DeletionTimestamp.IsZero() {
        // No special cleanup yet
        return ctrl.Result{}, nil
    }

    desiredBuildID := hc.Spec.Version
    if desiredBuildID == "" {
        desiredBuildID = hc.Spec.Source.Ref
    }
    if desiredBuildID == "" {
        desiredBuildID = "latest"
    }

    buildJobName := buildJobNamePrefix + hc.Name

    var job batchv1.Job
    err := r.Get(ctx, types.NamespacedName{Name: buildJobName, Namespace: hc.Namespace}, &job)
    if err != nil && errors.IsNotFound(err) {
        logger.Info("Creating build Job", "job", buildJobName)
        job = r.buildJobForHonse(&hc, buildJobName, desiredBuildID)
        if err := r.Create(ctx, &job); err != nil {
            return ctrl.Result{}, err
        }

        hc.Status.Phase = "Building"
        _ = r.Status().Update(ctx, &hc)
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    } else if err != nil {
        return ctrl.Result{}, err
    }

    if job.Status.Succeeded == 0 {
        if job.Status.Failed > 0 {
            hc.Status.Phase = "BuildFailed"
            _ = r.Status().Update(ctx, &hc)
            return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
        }
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }

    if hc.Status.LastBuildCommit != desiredBuildID {
        now := metav1.Now()
        hc.Status.LastBuildCommit = desiredBuildID
        hc.Status.LastBuildTime = &now
        hc.Status.Phase = "Built"
        _ = r.Status().Update(ctx, &hc)
    }

    if err := r.ensureServerRuntime(ctx, &hc, desiredBuildID); err != nil {
        return ctrl.Result{}, err
    }

    hc.Status.Phase = "Ready"
    _ = r.Status().Update(ctx, &hc)

    return ctrl.Result{}, nil
}

func (r *HonseClusterReconciler) buildJobForHonse(hc *honsev1alpha1.HonseCluster, jobName, tag string) batchv1.Job {
    backoff := int32(0)

    repo := hc.Spec.Source.RepoURL
    ref := hc.Spec.Source.Ref
    if ref == "" {
        ref = "main"
    }
    ctxBase := hc.Spec.Source.ContextBaseDir
    if ctxBase == "" {
        ctxBase = "."
    }

    registryHost := hc.Spec.Registry.Host
    repoPrefix := hc.Spec.Registry.RepositoryPrefix
    if repoPrefix == "" {
        repoPrefix = "honse"
    }

    var sb strings.Builder
    sb.WriteString("set -euo pipefail\n")
    sb.WriteString("echo 'Cloning repo'\n")
    sb.WriteString(fmt.Sprintf("git clone %q /workspace\n", repo))
    sb.WriteString("cd /workspace\n")
    sb.WriteString(fmt.Sprintf("git checkout %q || true\n", ref))
    sb.WriteString(fmt.Sprintf("cd %q\n", ctxBase))

    for _, c := range hc.Spec.Build.Components {
        imageRef := fmt.Sprintf("%s/%s/%s:%s", registryHost, repoPrefix, c.Name, tag)
        sb.WriteString(fmt.Sprintf("echo 'Building %s'\n", c.Name))
        sb.WriteString(fmt.Sprintf("buildah bud -t %q -f %q %q\n", imageRef, c.Dockerfile, c.ContextDir))
        sb.WriteString(fmt.Sprintf("echo 'Pushing %s'\n", imageRef))
        sb.WriteString(fmt.Sprintf("buildah push %q\n", imageRef))
    }

    script := sb.String()

    return batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      jobName,
            Namespace: hc.Namespace,
        },
        Spec: batchv1.JobSpec{
            BackoffLimit: &backoff,
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    RestartPolicy: corev1.RestartPolicyNever,
                    Containers: []corev1.Container{
                        {
                            Name:    "builder",
                            Image:   "quay.io/buildah/stable:latest",
                            Command: []string{"/bin/sh", "-c", script},
                            SecurityContext: &corev1.SecurityContext{
                                Privileged: func(b bool) *bool { return &b }(true),
                            },
                        },
                    },
                },
            },
        },
    }
}

func (r *HonseClusterReconciler) ensureServerRuntime(ctx context.Context, hc *honsev1alpha1.HonseCluster, tag string) error {
    var serverComp *honsev1alpha1.HonseBuildComponentSpec
    for _, c := range hc.Spec.Build.Components {
        if c.Name == "server" {
            cCopy := c
            serverComp = &cCopy
            break
        }
    }
    if serverComp == nil {
        return nil
    }

    registryHost := hc.Spec.Registry.Host
    repoPrefix := hc.Spec.Registry.RepositoryPrefix
    if repoPrefix == "" {
        repoPrefix = "honse"
    }
    imageRef := fmt.Sprintf("%s/%s/%s:%s", registryHost, repoPrefix, "server", tag)

    labels := map[string]string{
        "app":          "honse-server",
        "honsecluster": hc.Name,
    }

    var deploy appsv1.Deployment
    err := r.Get(ctx, types.NamespacedName{Name: "honse-server", Namespace: hc.Namespace}, &deploy)
    replicas := int32(1)

    if err != nil && errors.IsNotFound(err) {
        deploy = appsv1.Deployment{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "honse-server",
                Namespace: hc.Namespace,
                Labels:    labels,
            },
            Spec: appsv1.DeploymentSpec{
                Replicas: &replicas,
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
                                Name:  "server",
                                Image: imageRef,
                                Ports: []corev1.ContainerPort{
                                    {Name: "http", ContainerPort: 5000},
                                },
                            },
                        },
                    },
                },
            },
        }
        if err := controllerutil.SetControllerReference(hc, &deploy, r.Scheme); err != nil {
            return err
        }
        if err := r.Create(ctx, &deploy); err != nil {
            return err
        }
    } else if err != nil {
        return err
    } else {
        if len(deploy.Spec.Template.Spec.Containers) > 0 &&
            deploy.Spec.Template.Spec.Containers[0].Image != imageRef {
            deploy.Spec.Template.Spec.Containers[0].Image = imageRef
            if err := r.Update(ctx, &deploy); err != nil {
                return err
            }
        }
    }

    var svc corev1.Service
    err = r.Get(ctx, types.NamespacedName{Name: "honse-server", Namespace: hc.Namespace}, &svc)
    if err != nil && errors.IsNotFound(err) {
        svc = corev1.Service{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "honse-server",
                Namespace: hc.Namespace,
                Labels:    labels,
            },
            Spec: corev1.ServiceSpec{
                Selector: labels,
                Ports: []corev1.ServicePort{
                    {
                        Name:       "http",
                        Port:       5000,
                        TargetPort: intstr.FromInt(5000),
                    },
                },
            },
        }
        if err := controllerutil.SetControllerReference(hc, &svc, r.Scheme); err != nil {
            return err
        }
        if err := r.Create(ctx, &svc); err != nil {
            return err
        }
    } else if err != nil {
        return err
    }

    return nil
}

func (r *HonseClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&honsev1alpha1.HonseCluster{}).
        Owns(&batchv1.Job{}).
        Owns(&appsv1.Deployment{}).
        Owns(&corev1.Service{}).
        Complete(r)
}
