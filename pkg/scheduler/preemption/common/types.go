package preemptioncommon

// PreemptionPossibility 表示抢占模拟的结果。
type PreemptionPossibility int

const (
	NoCandidates PreemptionPossibility = iota // 未找到候选项。
	Preempt                                   // 找到了抢占目标。
	Reclaim                                   // 找到了抢占目标，且所有目标都在抢占 ClusterQueue 之外。
)
