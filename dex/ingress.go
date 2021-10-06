package dex

import (
	dexv1alpha1 "github.com/karavel-io/dex-operator/api/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"strings"
)

func Ingress(dex *dexv1alpha1.Dex) (networkingv1.Ingress, error) {
	ing := dex.Spec.Ingress
	labels := dex.Spec.InstanceLabels
	u, err := url.Parse(dex.Spec.PublicURL)
	if err != nil {
		return networkingv1.Ingress{}, err
	}

	if len(ing.Labels) > 0 {
		labels = ing.Labels
	}

	tls := make([]networkingv1.IngressTLS, 0)
	if ing.TLSEnabled {
		if ing.TLSSecretName == "" {
			ing.TLSSecretName = u.Host + "-tls"
		}

		tls = []networkingv1.IngressTLS{
			{
				Hosts:      []string{u.Host},
				SecretName: ing.TLSSecretName,
			},
		}
	}
	pathType := networkingv1.PathTypePrefix
	return networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        dex.ServiceName(),
			Namespace:   dex.Namespace,
			Labels:      labels,
			Annotations: ing.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			TLS: tls,
			Rules: []networkingv1.IngressRule{
				{
					Host: u.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/" + strings.TrimPrefix(u.Path, "/"),
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: dex.ServiceName(),
											Port: networkingv1.ServiceBackendPort{
												Name: "https",
											},
										},
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
