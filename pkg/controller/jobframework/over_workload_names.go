package jobframework

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

const (
	hashLength = 5
	// 253 is the maximal length for a CRD name. We need to subtract one for '-', and the hash length.
	maxPrefixLength = 252 - hashLength
)

func GetWorkloadNameForOwnerWithGVK(ownerName string, ownerUID types.UID, ownerGVK schema.GroupVersionKind) string {
	prefixedName := strings.ToLower(ownerGVK.Kind) + "-" + ownerName
	if len(prefixedName) > maxPrefixLength {
		prefixedName = prefixedName[:maxPrefixLength]
	}
	return prefixedName + "-" + getHash(ownerName, ownerUID, ownerGVK)[:hashLength]
}

func getHash(ownerName string, ownerUID types.UID, gvk schema.GroupVersionKind) string {
	h := sha1.New()
	h.Write([]byte(gvk.Kind))
	h.Write([]byte("\n"))
	h.Write([]byte(gvk.Group))
	h.Write([]byte("\n"))
	h.Write([]byte(ownerName))
	h.Write([]byte("\n"))
	h.Write([]byte(ownerUID))
	return hex.EncodeToString(h.Sum(nil))
}

func GetOwnerKey(ownerGVK schema.GroupVersionKind) string {
	return fmt.Sprintf(".metadata.ownerReferences[%s.%s]", ownerGVK.Group, ownerGVK.Kind)
}
