package dex

import (
	"crypto/sha256"
	"fmt"
	dexv1alpha1 "github.com/mikamai/dex-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Deployment(dex *dexv1alpha1.Dex, cm *v1.ConfigMap, sa *v1.ServiceAccount) appsv1.Deployment {
	labels := dex.Labels()
	csum := sha256.Sum256([]byte(cm.Data["config.yaml"]))
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-operated", dex.Name),
			Namespace:       dex.Namespace,
			OwnerReferences: []metav1.OwnerReference{dex.BuildOwnerReference()},
			Labels:          labels,
			Annotations: map[string]string{
				"config/checksum": fmt.Sprintf("%x", csum),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &dex.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: sa.Name,
					Containers: []v1.Container{
						{
							Name:    "dex",
							Image:   fmt.Sprintf("quay.io/dexidp/dex:v%s", dex.Version()),
							Command: []string{"/usr/local/bin/dex"},
							Args:    []string{"serve", "/etc/dex/cfg/config.yaml"},
							Ports: []v1.ContainerPort{
								{
									Name:          "https",
									ContainerPort: 5556,
									Protocol:      v1.ProtocolTCP,
								},
								{
									Name:          "grpc",
									ContainerPort: 5557,
									Protocol:      v1.ProtocolTCP,
								},
								{
									Name:          "metrics",
									ContainerPort: 5558,
									Protocol:      v1.ProtocolTCP,
								},
							},
							ReadinessProbe: &v1.Probe{
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/metrics",
										Port: intstr.IntOrString{IntVal: 5558},
									},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      5,
								FailureThreshold:    3,
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/dex/cfg",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: cm.Name,
									},
									Items: []v1.KeyToPath{
										{Key: "config.yaml", Path: "config.yaml"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
