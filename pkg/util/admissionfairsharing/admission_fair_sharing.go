package admissionfairsharing

import (
	corev1 "k8s.io/api/core/v1"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/features"
)

func ResourceWeights(cqAdmissionScope *kueue.AdmissionScope, afsConfig *config.AdmissionFairSharing) (bool, map[corev1.ResourceName]float64) {
	enableAdmissionFs, fsResWeights := false, make(map[corev1.ResourceName]float64)
	if features.Enabled(features.AdmissionFairSharing) && afsConfig != nil && cqAdmissionScope != nil && cqAdmissionScope.AdmissionMode == kueue.UsageBasedAdmissionFairSharing {
		enableAdmissionFs = true
		fsResWeights = afsConfig.ResourceWeights
	}
	return enableAdmissionFs, fsResWeights
}
