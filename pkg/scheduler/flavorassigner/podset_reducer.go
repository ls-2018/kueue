package flavorassigner

import (
	"sort"

	"k8s.io/utils/ptr"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// PodSetReducer 辅助结构体，用于从 PodSets[*].Count 逐步递减到 *PodSets[*].MinimumCount。
type PodSetReducer[R any] struct {
	podSets    []kueue.PodSet
	fullCounts []int32
	deltas     []int32
	totalDelta int32
	fits       func([]int32) (R, bool)
}

func NewPodSetReducer[R any](podSets []kueue.PodSet, fits func([]int32) (R, bool)) *PodSetReducer[R] {
	psr := &PodSetReducer[R]{
		podSets:    podSets,
		deltas:     make([]int32, len(podSets)),
		fullCounts: make([]int32, len(podSets)),
		fits:       fits,
	}

	for i := range psr.podSets {
		ps := &psr.podSets[i]
		psr.fullCounts[i] = ps.Count

		d := ps.Count - ptr.Deref(ps.MinCount, ps.Count)
		psr.deltas[i] = d
		psr.totalDelta += d
	}
	return psr
}

func fillPodSetSizesForSearchIndex(out, fullCounts, deltas []int32, upFactor int32, downFactor int32) {
	// 如果 len(out) < len(deltas) 会 panic
	for i, v := range deltas {
		tmp := int32(int64(v) * int64(upFactor) / int64(downFactor))
		out[i] = fullCounts[i] - tmp
	}
}

// Search 查找通过 fits() 的第一个最大计数集合，使用二分查找，因此最后一次调用 fits() 可能不是成功的。
// 如果未找到解决方案则返回 nil。
func (psr *PodSetReducer[R]) Search() (R, bool) {
	var lastGoodIdx int
	var lastR R

	if psr.totalDelta == 0 {
		return lastR, false
	}

	current := make([]int32, len(psr.podSets))
	idx := sort.Search(int(psr.totalDelta)+1, func(i int) bool {
		fillPodSetSizesForSearchIndex(current, psr.fullCounts, psr.deltas, int32(i), psr.totalDelta)
		r, f := psr.fits(current)
		if f {
			lastGoodIdx = i
			lastR = r
		}
		return f
	})
	return lastR, idx == lastGoodIdx
}
