package util

import (
	"regexp"
)

var K8sSelectorRegexp = regexp.MustCompile(`\s*([^!=\s]+)\s*(=|==|!=)\s*([^!=\s]+)\s*`)

// CheckLabelSelectorMismatch checks if selectors do not match the given labels
// It uses Kubernetes Label selectors with Equality-based requirement:
// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
func CheckLabelSelectorMismatch(labels map[string]string, selectors []string) bool {
	// Potentially refactor this and selector regexp with
	// https://github.com/kubernetes/apimachinery/blob/master/pkg/labels/selector.go
	for _, selector := range selectors {
		parts := K8sSelectorRegexp.FindStringSubmatch(selector)
		if parts != nil {
			selKey := parts[1]
			selEq := parts[2] == "=" || parts[2] == "=="
			selNotEq := parts[2] == "!="
			selValue := parts[3]
			v, ok := labels[selKey]
			if ok {
				if (selEq && v != selValue) || (selNotEq && v == selValue) {
					return true
				}
			}
		}
	}
	return false
}
