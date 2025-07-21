package node

import (
	corev1 "k8s.io/api/core/v1"
)

// NodeWrapper wraps a Node.
type NodeWrapper struct {
	corev1.Node
}

// MakeNode creates a wrapper for a Node

// Obj returns the inner Node.
func (n *NodeWrapper) Obj() *corev1.Node {
	return &n.Node
}

// Clone returns a deep copy of the NodeWrapper.
func (n *NodeWrapper) Clone() *NodeWrapper {
	return &NodeWrapper{Node: *n.Obj().DeepCopy()}
}

// Name updates the name of the node
func (n *NodeWrapper) Name(name string) *NodeWrapper {
	n.ObjectMeta.Name = name
	return n
}

// Label adds a label to the Node
func (n *NodeWrapper) Label(k, v string) *NodeWrapper {
	if n.Labels == nil {
		n.Labels = make(map[string]string)
	}
	n.Labels[k] = v
	return n
}

// StatusConditions appends the given status conditions to the Node.
func (n *NodeWrapper) StatusConditions(conditions ...corev1.NodeCondition) *NodeWrapper {
	n.Status.Conditions = append(n.Status.Conditions, conditions...)
	return n
}

// StatusAllocatable updates the allocatable resources of the Node.
func (n *NodeWrapper) StatusAllocatable(resources corev1.ResourceList) *NodeWrapper {
	if n.Status.Allocatable == nil {
		n.Status.Allocatable = make(corev1.ResourceList, len(resources))
	}
	for rName, rQuantity := range resources {
		n.Status.Allocatable[rName] = rQuantity
	}
	return n
}

// Taints appends the given taints to the Node.
func (n *NodeWrapper) Taints(taints ...corev1.Taint) *NodeWrapper {
	n.Spec.Taints = append(n.Spec.Taints, taints...)
	return n
}

// Ready sets the Node to a ready status condition
func (n *NodeWrapper) Ready() *NodeWrapper {
	n.StatusConditions(corev1.NodeCondition{
		Type:   corev1.NodeReady,
		Status: corev1.ConditionTrue,
	})
	return n
}

// NotReady sets the Node to a not ready status condition
func (n *NodeWrapper) NotReady() *NodeWrapper {
	n.StatusConditions(corev1.NodeCondition{
		Type:   corev1.NodeReady,
		Status: corev1.ConditionFalse,
	})
	return n
}

// Unschedulable sets the Node to an unschedulable state.
func (n *NodeWrapper) Unschedulable() *NodeWrapper {
	n.Spec.Unschedulable = true
	return n
}
