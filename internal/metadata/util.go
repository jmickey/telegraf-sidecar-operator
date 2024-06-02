package metadata

import "strings"

func GetAnnotationsWithPrefix(annotations map[string]string, prefix string) map[string]string {
	values := make(map[string]string)
	for k, v := range annotations {
		if strings.HasPrefix(k, prefix) {
			values[strings.TrimPrefix(k, prefix)] = v
		}
	}
	return values
}
