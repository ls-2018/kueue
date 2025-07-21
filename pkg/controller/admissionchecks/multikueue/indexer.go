package multikueue

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/util/admissioncheck"
)

const (
	UsingKubeConfigs             = "spec.kubeconfigs"
	UsingMultiKueueClusters      = "spec.multiKueueClusters"
	AdmissionCheckUsingConfigKey = "spec.multiKueueConfig"
)

var (
	configGVK = kueue.GroupVersion.WithKind("MultiKueueConfig")
)

func getIndexUsingKubeConfigs(configNamespace string) func(obj client.Object) []string {
	return func(obj client.Object) []string {
		cluster, isCluster := obj.(*kueue.MultiKueueCluster)
		if !isCluster {
			return nil
		}
		return []string{strings.Join([]string{configNamespace, cluster.Spec.KubeConfig.Location}, "/")}
	}
}

func indexUsingMultiKueueClusters(obj client.Object) []string {
	config, isConfig := obj.(*kueue.MultiKueueConfig)
	if !isConfig {
		return nil
	}
	return config.Spec.Clusters
}

func SetupIndexer(ctx context.Context, indexer client.FieldIndexer, configNamespace string) error {
	if err := indexer.IndexField(ctx, &kueue.MultiKueueCluster{}, UsingKubeConfigs, getIndexUsingKubeConfigs(configNamespace)); err != nil {
		return fmt.Errorf("setting index on clusters using kubeconfig: %w", err)
	}
	if err := indexer.IndexField(ctx, &kueue.MultiKueueConfig{}, UsingMultiKueueClusters, indexUsingMultiKueueClusters); err != nil {
		return fmt.Errorf("setting index on configs using clusters: %w", err)
	}
	if err := indexer.IndexField(ctx, &kueue.AdmissionCheck{}, AdmissionCheckUsingConfigKey, admissioncheck.IndexerByConfigFunction(kueue.MultiKueueControllerName, configGVK)); err != nil {
		return fmt.Errorf("setting index on admission checks config: %w", err)
	}
	return nil
}
