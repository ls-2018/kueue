package handlers

import (
	"context"
	"sort"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

// NamespacesWebSocketHandler streams namespaces that are related to Kueue
func NamespacesWebSocketHandler(dynamicClient dynamic.Interface) gin.HandlerFunc {
	return GenericWebSocketHandler(func() (any, error) {
		return fetchNamespaces(dynamicClient)
	})
}

// Fetch namespaces that have LocalQueues (Kueue-related namespaces)
func fetchNamespaces(dynamicClient dynamic.Interface) (any, error) {
	// First, get all LocalQueues to find namespaces that have them
	localQueues, err := dynamicClient.Resource(LocalQueuesGVR()).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Extract unique namespaces from LocalQueues
	namespaceSet := make(map[string]bool)
	for _, lq := range localQueues.Items {
		namespace := lq.GetNamespace()
		if namespace != "" {
			namespaceSet[namespace] = true
		}
	}

	// If no LocalQueues found, return empty result with proper structure
	if len(namespaceSet) == 0 {
		return map[string]any{
			"namespaces": []string{},
		}, nil
	}

	// Convert namespace set to a slice of strings (just the names)
	var namespaceNames []string
	for namespaceName := range namespaceSet {
		namespaceNames = append(namespaceNames, namespaceName)
	}

	// Sort namespaces alphabetically for consistent ordering
	sort.Strings(namespaceNames)

	// Return in the same format as other endpoints
	result := map[string]any{
		"namespaces": namespaceNames,
	}

	return result, nil
}
