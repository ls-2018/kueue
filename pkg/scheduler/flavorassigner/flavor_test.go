package flavorassigner

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
	"testing"
)

func TestMatch(t *testing.T) {
	fmt.Println(corev1helpers.FindMatchingUntoleratedTaint([]corev1.Taint{
		corev1.Taint{
			Key:    "foo",
			Value:  "bar",
			Effect: corev1.TaintEffectNoSchedule,
		},
	}, []corev1.Toleration{
		corev1.Toleration{
			Key:    "foo",
			Value:  "bar",
			Effect: corev1.TaintEffectNoSchedule,
		},
	}, func(t *corev1.Taint) bool {
		return t.Effect == corev1.TaintEffectNoSchedule || t.Effect == corev1.TaintEffectNoExecute
	}))
}
