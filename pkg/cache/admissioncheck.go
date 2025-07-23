package cache

// AdmissionCheck 结构体用于描述准入检查的状态和控制器。
type AdmissionCheck struct {
	Active     bool
	Controller string
}
