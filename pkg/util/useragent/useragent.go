package useragent

import (
	"fmt"
	"runtime"
	"strings"

	"sigs.k8s.io/kueue/pkg/constants"
	"sigs.k8s.io/kueue/pkg/version"
)

// adjustVersion strips "alpha", "beta", etc. from version in form
// major.minor.patch-[alpha|beta|etc].
func adjustVersion(v string) string {
	if len(v) == 0 {
		return "unknown"
	}
	seg := strings.SplitN(v, "-", 2)
	return seg[0]
}

// adjustCommit returns sufficient significant figures of the commit's git hash.
func adjustCommit(c string) string {
	if len(c) == 0 {
		return "unknown"
	}
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// Default returns User-Agent string built from static global vars.
func Default() string {
	return fmt.Sprintf("%s/%s (%s/%s) %s",
		constants.KueueName,
		adjustVersion(version.GitVersion),
		runtime.GOOS,
		runtime.GOARCH,
		adjustCommit(version.GitCommit))
}
