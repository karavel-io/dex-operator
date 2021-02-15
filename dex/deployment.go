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

func Service(dex *dexv1alpha1.Dex) v1.Service {
	labels := dex.Labels()
	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dex.ServiceName(),
			Namespace: dex.Namespace,
			Labels:    labels,
		},
		Spec: v1.ServiceSpec{
			Selector: labels,
			Type:     v1.ServiceTypeClusterIP,
			Ports: []v1.ServicePort{
				{
					Name:       "https",
					Port:       5556,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromString("https"),
				},
				{
					Name:       "grpc",
					Port:       5557,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromString("grpc"),
				},
			},
		},
	}
}

func MetricsService(dex *dexv1alpha1.Dex) v1.Service {
	labels := dex.Labels()
	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dex.ServiceName() + "-metrics",
			Namespace: dex.Namespace,
			Labels:    labels,
		},
		Spec: v1.ServiceSpec{
			Selector: labels,
			Type:     v1.ServiceTypeClusterIP,
			Ports: []v1.ServicePort{
				{
					Name:       "metrics",
					Port:       5558,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromString("metrics"),
				},
			},
		},
	}
}

func Deployment(dex *dexv1alpha1.Dex, cm *v1.ConfigMap, sa *v1.ServiceAccount) appsv1.Deployment {
	csum := fmt.Sprintf("%x", sha256.Sum256([]byte(cm.Data["config.yaml"])))
	labels := dex.Labels()
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dex.ServiceName(),
			Namespace: dex.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &dex.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"config/checksum": csum,
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: sa.Name,
					Containers: []v1.Container{
						{
							Name:    "dex",
							Image:   fmt.Sprintf("quay.io/dexidp/dex:v%s", dex.Version()),
							Command: []string{"/usr/local/bin/dex"},
							Args:    []string{"serve", "/etc/dex/cfg/config.yaml"},
							EnvFrom: dex.Spec.EnvFrom,
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
