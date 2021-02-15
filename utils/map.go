package utils

func ShallowCopyLabels(m map[string]string) map[string]string {
	labels := make(map[string]string)
	for k, v := range m {
		labels[k] = v
	}
	return labels
}
