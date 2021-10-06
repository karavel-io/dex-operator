package dex

import (
	"crypto/sha256"
	"fmt"
	dexv1alpha1 "github.com/karavel-io/dex-operator/api/v1alpha1"
	"github.com/karavel-io/dex-operator/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	InstanceMarkerLabel = "dex.karavel.io/instance"
	PortHttps           = 5556
	PortGrpc            = 5557
)

func Service(dex *dexv1alpha1.Dex) (v1.Service, string) {
	labels := utils.ShallowCopyLabels(dex.Spec.InstanceLabels)
	labels[InstanceMarkerLabel] = dex.Name
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
					Port:       PortHttps,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromString("https"),
				},
				{
					Name:       "grpc",
					Port:       PortGrpc,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromString("grpc"),
				},
			},
		},
	}, fmt.Sprintf("%s.%s:%d", dex.ServiceName(), dex.Namespace, PortGrpc)
}

func MetricsService(dex *dexv1alpha1.Dex) v1.Service {
	labels := utils.ShallowCopyLabels(dex.Spec.InstanceLabels)
	labels[InstanceMarkerLabel] = dex.Name
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
	labels := utils.ShallowCopyLabels(dex.Spec.InstanceLabels)
	labels[InstanceMarkerLabel] = dex.Name

	if dex.Spec.Resources.Limits == nil {
		dex.Spec.Resources.Limits = map[v1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse("200m"),
			"memory": resource.MustParse("200Mi"),
		}
	}

	if dex.Spec.Resources.Requests == nil {
		dex.Spec.Resources.Requests = map[v1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse("100m"),
			"memory": resource.MustParse("100Mi"),
		}
	}

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
					ImagePullSecrets:   dex.Spec.ImagePullSecrets,
					ServiceAccountName: sa.Name,
					Containers: []v1.Container{
						{
							Name:    "dex",
							Image:   dex.Spec.Image,
							Command: []string{"dex"},
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
							Resources: dex.Spec.Resources,
						},
					},
					Affinity:                  dex.Spec.Affinity,
					NodeSelector:              dex.Spec.NodeSelector,
					Tolerations:               dex.Spec.Tolerations,
					TopologySpreadConstraints: dex.Spec.TopologySpreadConstraints,
					SecurityContext:           dex.Spec.SecurityContext,
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
