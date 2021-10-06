package dex

import (
	dexv1alpha1 "github.com/karavel-io/dex-operator/api/v1alpha1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"net/url"
	"strings"
)

func Ingress(dex *dexv1alpha1.Dex) (networkingv1beta1.Ingress, error) {
	ing := dex.Spec.Ingress
	labels := dex.Spec.InstanceLabels
	u, err := url.Parse(dex.Spec.PublicURL)
	if err != nil {
		return networkingv1beta1.Ingress{}, err
	}

	if len(ing.Labels) > 0 {
		labels = ing.Labels
	}

	tls := make([]networkingv1beta1.IngressTLS, 0)
	if ing.TLSEnabled {
		if ing.TLSSecretName == "" {
			ing.TLSSecretName = u.Host + "-tls"
		}

		tls = []networkingv1beta1.IngressTLS{
			{
				Hosts:      []string{u.Host},
				SecretName: ing.TLSSecretName,
			},
		}
	}
	return networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        dex.ServiceName(),
			Namespace:   dex.Namespace,
			Labels:      labels,
			Annotations: ing.Annotations,
		},
		Spec: networkingv1beta1.IngressSpec{
			TLS: tls,
			Rules: []networkingv1beta1.IngressRule{
				{
					Host: u.Host,
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								{
									Path: "/" + strings.TrimPrefix(u.Path, "/"),
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: dex.ServiceName(),
										ServicePort: intstr.FromString("https"),
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}
