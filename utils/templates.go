package utils

import (
	"gopkg.in/yaml.v3"
	"strings"
)

func ToYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}

	return strings.TrimSuffix(string(data), "\n")
}
