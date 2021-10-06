package dex

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	dexv1alpha1 "github.com/karavel-io/dex-operator/api/v1alpha1"
	"github.com/karavel-io/dex-operator/utils"
	"html/template"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

var configTpl = `
issuer: {{ .PublicURL }}
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
{{- if .Config }}
    config:
{{ .Config | toYaml | indent 6 }}
{{- end }}
{{- end }}

oauth2:
  skipApprovalScreen: true

enablePasswordDB: false
`

type Connector struct {
	Type   string
	ID     string
	Name   string
	Config interface{}
}
type Config struct {
	PublicURL  string
	Connectors []Connector
}

func ConfigMap(dex *dexv1alpha1.Dex) (v1.ConfigMap, error) {
	f := sprig.FuncMap()
	f["toYaml"] = utils.ToYAML

	cfg := Config{
		PublicURL:  dex.Spec.PublicURL,
		Connectors: make([]Connector, len(dex.Spec.Connectors)),
	}
	for i, c := range dex.Spec.Connectors {
		cc := Connector{
			Type:   c.Type,
			ID:     c.ID,
			Name:   c.Name,
			Config: nil,
		}
		if len(c.Config.Raw) > 0 {
			err := json.Unmarshal(c.Config.Raw, &cc.Config)
			if err != nil {
				return v1.ConfigMap{}, err
			}
		}

		cfg.Connectors[i] = cc
	}

	tpl, err := template.New("config").Funcs(f).Parse(configTpl)
	if err != nil {
		return v1.ConfigMap{}, err
	}
	var s bytes.Buffer
	if err := tpl.Execute(&s, cfg); err != nil {
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
