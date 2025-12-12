package config

import "strings"

func containsAnyHost(hostname string, expectedHosts []string) bool {
	for _, expectedHost := range expectedHosts {
		if strings.Contains(hostname, expectedHost) {
			return true
		}
	}

	return false
}
