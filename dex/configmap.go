package dex

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	dexv1alpha1 "github.com/mikamai/dex-operator/api/v1alpha1"
	"github.com/mikamai/dex-operator/utils"
	"html/template"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var configTpl = `
issuer: https://{{ .PublicHost }}
storage:
  type: kubernetes
  config:
    inCluster: true
web:
  http: 0.0.0.0:5556
  # https: 0.0.0.0:5556
  # tlsCert: /etc/dex/tls/tls.crt
  # tlsKey: /etc/dex/tls/tls.key
grpc:
  addr: 0.0.0.0:5557
  # tlsCert: /etc/dex/tls/tls.crt
  # tlsKey: /etc/dex/tls/tls.key
telemetry:
  http: 0.0.0.0:5558
logger:
  level: "info"
  format: "json"
connectors:
{{- range .Connectors }}
  - type: {{ .Type }}
    id: {{ .ID }}
    name: {{ .Name }}
    config:
{{ .Config | indent 6 }}
{{- end }}

oauth2:
  skipApprovalScreen: true

enablePasswordDB: false
`

func ConfigMap(dex *dexv1alpha1.Dex) (v1.ConfigMap, error) {
	f := sprig.FuncMap()
	f["toYaml"] = utils.ToYAML

	tpl, err := template.New("config").Funcs(f).Parse(configTpl)
	if err != nil {
		return v1.ConfigMap{}, err
	}
	var s bytes.Buffer
	if err := tpl.Execute(&s, dex.Spec); err != nil {
		return v1.ConfigMap{}, err
	}

	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", dex.Name),
			Namespace: dex.Namespace,
			Labels:    dex.Spec.InstanceLabels,
		},
		Data: map[string]string{
			"config.yaml": s.String(),
		},
	}, nil
}
