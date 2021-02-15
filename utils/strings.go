package utils

func ContainsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func PopString(haystack []string, needle string) (rest []string) {
	for _, s := range haystack {
		if s == needle {
			continue
		}
		rest = append(rest, s)
	}
	return
}
