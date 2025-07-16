package api

import (
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxEventMsgSize     = 1024
	maxConditionMsgSize = 32 * 1024
)

// TruncateEventMessage truncates a message if it hits the maxEventMessage.
func TruncateEventMessage(message string) string {
	return truncateMessage(message, maxEventMsgSize)
}

func TruncateConditionMessage(message string) string {
	return truncateMessage(message, maxConditionMsgSize)
}

// truncateMessage truncates a message if it hits the NoteLengthLimit.
func truncateMessage(message string, limit int) string {
	if len(message) <= limit {
		return message
	}
	suffix := " ..."
	return message[:limit-len(suffix)] + suffix
}

// CloneObjectMetaForCreation creates a copy of the provided ObjectMeta containing
// only the name, namespace, labels and annotations
func CloneObjectMetaForCreation(orig *metav1.ObjectMeta) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        orig.Name,
		Namespace:   orig.Namespace,
		Labels:      maps.Clone(orig.Labels),
		Annotations: maps.Clone(orig.Annotations),
	}
}
