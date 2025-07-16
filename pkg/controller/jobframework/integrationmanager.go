package jobframework

import (
	"context"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"slices"
	"sort"
	"sync"
)

var (
	errDuplicateFrameworkName = errors.New("duplicate framework name")
	errMissingMandatoryField  = errors.New("mandatory field missing")
	errFrameworkNameFormat    = errors.New("misformatted external framework name")

	errIntegrationNotFound             = errors.New("integration not found")
	errDependencyIntegrationNotEnabled = errors.New("integration not enabled")
)

type JobReconcilerInterface interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager) error
}

type ReconcilerFactory func(client client.Client, record record.EventRecorder, opts ...Option) JobReconcilerInterface

// IntegrationCallbacks 组织了一组用于集成新框架的回调。
type IntegrationCallbacks struct {
	// NewJob 创建一个新的作业实例
	NewJob func() GenericJob
	// GVK 保存作业的 schema 信息（此回调为可选）
	GVK schema.GroupVersionKind
	// NewReconciler 创建一个新的调谐器
	NewReconciler ReconcilerFactory
	// NewAdditionalReconcilers 创建额外的调谐器（此回调为可选）
	NewAdditionalReconcilers []ReconcilerFactory
	// SetupWebhook 使用控制器管理器设置框架的 webhook
	SetupWebhook func(mgr ctrl.Manager, opts ...Option) error
	// JobType 保存由集成 webhook 管理的对象类型
	JobType runtime.Object
	// SetupIndexes 向控制器管理器注册任何额外的索引（此回调为可选）
	SetupIndexes func(ctx context.Context, indexer client.FieldIndexer) error
	// AddToScheme 向控制器管理器的 scheme 添加任何额外的类型（此回调为可选）
	AddToScheme func(s *runtime.Scheme) error
	// CanSupportIntegration 如果集成满足任何额外条件（如 Kubernetes 版本）则返回 true。
	CanSupportIntegration func(opts ...Option) (bool, error)
	// 作业的 MultiKueue 适配器（可选）
	MultiKueueAdapter MultiKueueAdapter
	// 需要与当前集成一起启用的集成列表。
	DependencyList []string
}

func (i *IntegrationCallbacks) getGVK() schema.GroupVersionKind {
	if i.NewJob != nil {
		return i.NewJob().GVK()
	}
	return i.GVK
}

func (i *IntegrationCallbacks) matchingGVK(gvk schema.GroupVersionKind) bool {
	return i.getGVK() == gvk
}

func (i *IntegrationCallbacks) matchingOwnerReference(ownerRef *metav1.OwnerReference) bool {
	return ownerReferenceMatchingGVK(ownerRef, i.getGVK())
}

type integrationManager struct {
	names                []string
	integrations         map[string]IntegrationCallbacks
	enabledIntegrations  set.Set[string]
	externalIntegrations map[string]runtime.Object
	mu                   sync.RWMutex
}

var manager integrationManager

func (m *integrationManager) get(name string) (IntegrationCallbacks, bool) {
	cb, f := m.integrations[name]
	return cb, f
}

func (m *integrationManager) getExternal(kindArg string) (runtime.Object, bool) {
	jt, f := m.externalIntegrations[kindArg]
	return jt, f
}

func (m *integrationManager) getEnabledIntegrations() set.Set[string] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabledIntegrations.Clone()
}

func (m *integrationManager) getList() []string {
	ret := make([]string, len(m.names))
	copy(ret, m.names)
	sort.Strings(ret)
	return ret
}

func (m *integrationManager) isKnownOwner(ownerRef *metav1.OwnerReference) bool {
	for _, cbs := range m.integrations {
		if cbs.matchingOwnerReference(ownerRef) {
			return true
		}
	}
	for _, jt := range m.externalIntegrations {
		if ownerReferenceMatchingGVK(ownerRef, jt.GetObjectKind().GroupVersionKind()) {
			return true
		}
	}
	// ReplicaSet is an interim owner from Pod to Deployment. We call it known
	// so that the users don't need to list	it explicitly in their configs.
	// Note that Kueue provides RBAC permissions allowing for traversal over it.
	// ReplicaSet 是 Pod 到 Deployment 之间的中间所有者。我们将其视为已知，
	// 这样用户就不需要在配置中显式列出它。
	// 注意 Kueue 提供了允许遍历它的 RBAC 权限。
	return ownerRef.Kind == "ReplicaSet" && ownerRef.APIVersion == "apps/v1"
}

func (m *integrationManager) getJobTypeForOwner(ownerRef *metav1.OwnerReference) runtime.Object {
	for jobKey := range m.getEnabledIntegrations() {
		cbs, found := m.integrations[jobKey]
		if found && cbs.matchingOwnerReference(ownerRef) {
			return cbs.JobType
		}
	}
	for _, jt := range m.externalIntegrations {
		if ownerReferenceMatchingGVK(ownerRef, jt.GetObjectKind().GroupVersionKind()) {
			return jt
		}
	}

	return nil
}

// ForEachIntegration loops through the registered list of frameworks calling f,
// if at any point f returns an error the loop is stopped and that error is returned.
// ForEachIntegration 遍历已注册的框架列表并调用 f，
// 如果 f 在任何时候返回错误，则循环停止并返回该错误。
func ForEachIntegration(f func(name string, cb IntegrationCallbacks) error) error {
	return manager.forEach(f)
}

// GetIntegration looks-up the framework identified by name in the currently registered
// list of frameworks returning its callbacks and true if found.
// GetIntegration 在当前已注册的框架列表中查找由名称标识的框架，
// 如果找到则返回其回调和 true。
func GetIntegration(name string) (IntegrationCallbacks, bool) {
	return manager.get(name)
}

// GetIntegrationByGVK looks-up the framework identified by GroupVersionKind in the currently
// registered list of frameworks returning its callbacks and true if found.
// GetIntegrationByGVK 在当前已注册的框架列表中查找由 GroupVersionKind 标识的框架，
// 如果找到则返回其回调和 true。
func GetIntegrationByGVK(gvk schema.GroupVersionKind) (IntegrationCallbacks, bool) {
	for _, name := range manager.getList() {
		integration, ok := GetIntegration(name)
		if ok && integration.matchingGVK(gvk) {
			return integration, true
		}
	}
	return IntegrationCallbacks{}, false
}

func ownerReferenceMatchingGVK(ownerRef *metav1.OwnerReference, gvk schema.GroupVersionKind) bool {
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	return ownerRef.APIVersion == apiVersion && ownerRef.Kind == kind
}

// GetIntegrationsList returns the list of currently registered frameworks.
// GetIntegrationsList 返回当前已注册的框架列表。
func GetIntegrationsList() []string {
	return manager.getList()
}

// IsOwnerManagedByKueueForObject returns true if the provided object has an owner,
// and this owner can be managed by Kueue.
// IsOwnerManagedByKueueForObject 如果提供的对象有所有者，且该所有者可以被 Kueue 管理，则返回 true。
func IsOwnerManagedByKueueForObject(obj client.Object) bool {
	if owner := metav1.GetControllerOf(obj); owner != nil {
		return manager.getJobTypeForOwner(owner) != nil
	}
	return false
}

// getEmptyOwnerObject returns an empty object of the owner's type,
// returns nil if the owner is not manageable by kueue.
// getEmptyOwnerObject 返回所有者类型的空对象，
// 如果所有者不能被 kueue 管理则返回 nil。
func getEmptyOwnerObject(owner *metav1.OwnerReference) client.Object {
	if jt := manager.getJobTypeForOwner(owner); jt != nil {
		return jt.DeepCopyObject().(client.Object)
	}
	return nil
}

// GetMultiKueueAdapters returns the map containing the MultiKueue adapters for the
// registered and enabled integrations.
// An error is returned if more then one adapter is registers for one object type.
// GetMultiKueueAdapters 返回包含已注册和启用集成的 MultiKueue 适配器的映射。
// 如果为一个对象类型注册了多个适配器，则返回错误。
func GetMultiKueueAdapters(enabledIntegrations sets.Set[string]) (map[string]MultiKueueAdapter, error) {
	ret := map[string]MultiKueueAdapter{}
	if err := manager.forEach(func(intName string, cb IntegrationCallbacks) error {
		if cb.MultiKueueAdapter != nil && enabledIntegrations.Has(intName) {
			gvk := cb.MultiKueueAdapter.GVK().String()
			if _, found := ret[gvk]; found {
				return fmt.Errorf("multiple adapters for GVK: %q", gvk)
			}
			ret[gvk] = cb.MultiKueueAdapter
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

// RegisterIntegration registers a new framework, returns an error when
// attempting to register multiple frameworks with the same name or if a
// mandatory callback is missing.
// RegisterIntegration 注册一个新框架，
// 当尝试注册同名框架或缺少必需回调时返回错误。
func RegisterIntegration(name string, cb IntegrationCallbacks) error {
	return manager.register(name, cb)
}
func (m *integrationManager) register(name string, cb IntegrationCallbacks) error {
	if m.integrations == nil {
		m.integrations = make(map[string]IntegrationCallbacks)
	}
	if _, exists := m.integrations[name]; exists {
		return fmt.Errorf("%w %q", errDuplicateFrameworkName, name)
	}

	if cb.NewReconciler == nil {
		return fmt.Errorf("%w \"NewReconciler\" for %q", errMissingMandatoryField, name)
	}

	if cb.SetupWebhook == nil {
		return fmt.Errorf("%w \"SetupWebhook\" for %q", errMissingMandatoryField, name)
	}

	if cb.JobType == nil {
		return fmt.Errorf("%w \"WebhookType\" for %q", errMissingMandatoryField, name)
	}

	m.integrations[name] = cb
	m.names = append(m.names, name)

	return nil
}

func (m *integrationManager) checkEnabledListDependencies(enabledSet sets.Set[string]) error {
	enabled := enabledSet.UnsortedList()
	slices.Sort(enabled)
	for _, integration := range enabled {
		cbs, found := m.integrations[integration]
		if !found {
			return fmt.Errorf("%q %w", integration, errIntegrationNotFound)
		}
		for _, dep := range cbs.DependencyList {
			if !enabledSet.Has(dep) {
				return fmt.Errorf("%q %w %q", integration, errDependencyIntegrationNotEnabled, dep)
			}
		}
	}
	return nil
}

// RegisterExternalJobType registers an external job type by kindArg.
// RegisterExternalJobType 通过 kindArg 注册一个外部作业类型。
func RegisterExternalJobType(kindArg string) error {
	return manager.registerExternal(kindArg)
}
func (m *integrationManager) registerExternal(kindArg string) error {
	if m.externalIntegrations == nil {
		m.externalIntegrations = make(map[string]runtime.Object)
	}

	gvk, _ := schema.ParseKindArg(kindArg)
	if gvk == nil {
		return fmt.Errorf("%w %q", errFrameworkNameFormat, kindArg)
	}
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	jobType := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       kind,
		},
	}

	m.externalIntegrations[kindArg] = jobType

	return nil
}

// forEach loops through the registered list of frameworks calling f.
// forEach 遍历已注册的框架列表并调用 f。
func (m *integrationManager) forEach(f func(name string, cb IntegrationCallbacks) error) error {
	for _, name := range m.names {
		if err := f(name, m.integrations[name]); err != nil {
			return err
		}
	}
	return nil
}
func (m *integrationManager) enableIntegration(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.enabledIntegrations == nil {
		m.enabledIntegrations = set.New(name)
	} else {
		m.enabledIntegrations.Insert(name)
	}
}
