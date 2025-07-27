package tas

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
)

type TopologyDomainID string

func DomainID(levelValues []string) TopologyDomainID {
	return TopologyDomainID(strings.Join(levelValues, ","))
}

func NodeLabelsFromKeysAndValues(keys, values []string) map[string]string {
	result := make(map[string]string, len(keys))
	for i := range keys {
		result[keys[i]] = values[i]
	}
	return result
}

func LevelValues(levelKeys []string, objectLabels map[string]string) []string {
	levelValues := make([]string, len(levelKeys))
	for levelIdx, levelKey := range levelKeys {
		levelValues[levelIdx] = objectLabels[levelKey]
	}
	return levelValues
}

func Levels(topology *kueuealpha.Topology) []string {
	result := make([]string, len(topology.Spec.Levels))
	for i, level := range topology.Spec.Levels {
		result[i] = level.NodeLabel
	}
	return result
}

func IsNodeStatusConditionTrue(conditions []corev1.NodeCondition, conditionType corev1.NodeConditionType) bool {
	for _, cond := range conditions {
		if cond.Type == conditionType {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

// IsLowestLevelHostname checks if the lowest (last) level in the provided topology levels is node
func IsLowestLevelHostname(levels []string) bool {
	return levels[len(levels)-1] == corev1.LabelHostname
}
