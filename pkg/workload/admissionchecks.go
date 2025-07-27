package workload

import (
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/clock"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// FindAdmissionCheck - 返回一个指向在 checks 中找到的、由 checkName 标识的检查的指针。
func FindAdmissionCheck(checks []kueue.AdmissionCheckState, checkName kueue.AdmissionCheckReference) *kueue.AdmissionCheckState {
	for i := range checks {
		if checks[i].Name == checkName {
			return &checks[i]
		}
	}
	return nil
}

// ResetChecksOnEviction 将所有 AdmissionChecks 重置为 Pending
func ResetChecksOnEviction(w *kueue.Workload, now time.Time) bool {
	checks := w.Status.AdmissionChecks
	updated := false
	for i := range checks {
		if checks[i].State == kueue.CheckStatePending {
			continue
		}
		checks[i] = kueue.AdmissionCheckState{
			Name:               checks[i].Name,
			State:              kueue.CheckStatePending,
			LastTransitionTime: metav1.NewTime(now),
			Message:            "Reset to Pending after eviction. Previously: " + string(checks[i].State),
		}
		updated = true
	}
	return updated
}

// RejectedChecks 返回拒绝的 Admission 检查列表
func RejectedChecks(wl *kueue.Workload) []kueue.AdmissionCheckState {
	rejectedChecks := make([]kueue.AdmissionCheckState, 0, len(wl.Status.AdmissionChecks))
	for i := range wl.Status.AdmissionChecks {
		ac := wl.Status.AdmissionChecks[i]
		if ac.State == kueue.CheckStateRejected {
			rejectedChecks = append(rejectedChecks, ac)
		}
	}
	return rejectedChecks
}

// HasAllChecksReady 返回 true 如果工作负载的所有检查都已准备就绪。
func HasAllChecksReady(wl *kueue.Workload) bool {
	for i := range wl.Status.AdmissionChecks {
		if wl.Status.AdmissionChecks[i].State != kueue.CheckStateReady {
			return false
		}
	}
	return true
}

// HasRetryChecks 返回 true 如果工作负载的任何检查是 Retry。
func HasRetryChecks(wl *kueue.Workload) bool {
	for i := range wl.Status.AdmissionChecks {
		state := wl.Status.AdmissionChecks[i].State
		if state == kueue.CheckStateRetry {
			return true
		}
	}
	return false
}

// HasRejectedChecks 返回 true 如果工作负载的任何检查是 Rejected。
func HasRejectedChecks(wl *kueue.Workload) bool {
	for i := range wl.Status.AdmissionChecks {
		state := wl.Status.AdmissionChecks[i].State
		if state == kueue.CheckStateRejected {
			return true
		}
	}
	return false
}

// HasAllChecks 返回 true 如果工作负载中存在所有 mustHaveChecks。
func HasAllChecks(wl *kueue.Workload, mustHaveChecks sets.Set[kueue.AdmissionCheckReference]) bool {
	if mustHaveChecks.Len() == 0 {
		return true
	}

	if mustHaveChecks.Len() > len(wl.Status.AdmissionChecks) {
		return false
	}
	// 担心 wl.Status.AdmissionChecks 会有重复的
	mustHaveChecks = mustHaveChecks.Clone()
	for i := range wl.Status.AdmissionChecks {
		mustHaveChecks.Delete(wl.Status.AdmissionChecks[i].Name)
	}
	return mustHaveChecks.Len() == 0
}

// SyncAdmittedCondition 同步 Admitted 条件的当前状态，基于 QuotaReserved、AdmissionChecks 和 DelayedTopologyRequests 的状态。
// 返回 true 表示有任何更改。
func SyncAdmittedCondition(w *kueue.Workload, now time.Time) bool {
	hasReservation := HasQuotaReservation(w)
	hasAllChecksReady := HasAllChecksReady(w)
	isAdmitted := IsAdmitted(w)
	hasAllTopologyAssignmentsReady := !HasTopologyAssignmentsPending(w)

	if isAdmitted == (hasReservation && hasAllChecksReady && hasAllTopologyAssignmentsReady) {
		return false
	}
	newCondition := metav1.Condition{
		Type:               kueue.WorkloadAdmitted,
		Status:             metav1.ConditionTrue,
		Reason:             "Admitted",
		Message:            "The workload is admitted",
		ObservedGeneration: w.Generation,
	}
	switch {
	case !hasReservation && !hasAllChecksReady:
		newCondition.Status = metav1.ConditionFalse
		newCondition.Reason = "NoReservationUnsatisfiedChecks"
		newCondition.Message = "The workload has no reservation and not all checks ready"
	case !hasReservation:
		newCondition.Status = metav1.ConditionFalse
		newCondition.Reason = "NoReservation"
		newCondition.Message = "The workload has no reservation"
	case !hasAllChecksReady:
		newCondition.Status = metav1.ConditionFalse
		newCondition.Reason = "UnsatisfiedChecks"
		newCondition.Message = "The workload has not all checks ready"
	case !hasAllTopologyAssignmentsReady:
		newCondition.Status = metav1.ConditionFalse
		newCondition.Reason = "PendingDelayedTopologyRequests"
		newCondition.Message = "There are pending delayed topology requests"
	}

	// Accumulate the admitted time if needed
	if isAdmitted && newCondition.Status == metav1.ConditionFalse {
		oldCondition := apimeta.FindStatusCondition(w.Status.Conditions, kueue.WorkloadAdmitted)
		// in practice the oldCondition cannot be nil, however we should try to avoid nil ptr deref.
		if oldCondition != nil {
			d := int32(now.Sub(oldCondition.LastTransitionTime.Time).Seconds())
			if w.Status.AccumulatedPastExexcutionTimeSeconds != nil {
				*w.Status.AccumulatedPastExexcutionTimeSeconds += d
			} else {
				w.Status.AccumulatedPastExexcutionTimeSeconds = &d
			}
		}
	}
	return apimeta.SetStatusCondition(&w.Status.Conditions, newCondition)
}

// SetAdmissionCheckState - 在提供的 checks 列表中添加或更新 newCheck。
func SetAdmissionCheckState(checks *[]kueue.AdmissionCheckState, newCheck kueue.AdmissionCheckState, clock clock.Clock) {
	if checks == nil {
		return
	}
	existingCondition := FindAdmissionCheck(*checks, newCheck.Name)
	if existingCondition == nil {
		if newCheck.LastTransitionTime.IsZero() {
			newCheck.LastTransitionTime = metav1.NewTime(clock.Now())
		}
		*checks = append(*checks, newCheck)
		return
	}

	if existingCondition.State != newCheck.State {
		existingCondition.State = newCheck.State
		if !newCheck.LastTransitionTime.IsZero() {
			existingCondition.LastTransitionTime = newCheck.LastTransitionTime
		} else {
			existingCondition.LastTransitionTime = metav1.NewTime(clock.Now())
		}
	}
	existingCondition.Message = newCheck.Message
	existingCondition.PodSetUpdates = newCheck.PodSetUpdates
}
