package clean

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/kueue/client-go/clientset/versioned"
)

func Clean(kContext string) {
	cfg, err := config.GetConfigWithContext(kContext)
	if err != nil {
		panic(err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	list, err := kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, ns := range list.Items {
		if strings.HasPrefix(ns.Name, "multikueue") {
			kubeClient.CoreV1().Namespaces().Delete(context.Background(), ns.Name, metav1.DeleteOptions{})
			ns.Finalizers = nil
			kubeClient.CoreV1().Namespaces().Update(context.Background(), &ns, metav1.UpdateOptions{})
		}
	}

	client, err := versioned.NewForConfig(cfg)

	{
		list, err := client.KueueV1alpha1().Cohorts("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, cohort := range list.Items {
			client.KueueV1alpha1().Cohorts(cohort.Namespace).Delete(context.Background(), cohort.Name, metav1.DeleteOptions{})
			cohort.Finalizers = nil
			client.KueueV1alpha1().Cohorts(cohort.Namespace).Update(context.Background(), &cohort, metav1.UpdateOptions{})
		}
	}
	{
		list, err := client.KueueV1beta1().AdmissionChecks().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, item := range list.Items {
			client.KueueV1beta1().AdmissionChecks().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
			item.Finalizers = nil
			client.KueueV1beta1().AdmissionChecks().Update(context.Background(), &item, metav1.UpdateOptions{})
		}
	}
	{
		list, err := client.KueueV1beta1().ClusterQueues().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, item := range list.Items {
			client.KueueV1beta1().ClusterQueues().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
			item.Finalizers = nil
			client.KueueV1beta1().ClusterQueues().Update(context.Background(), &item, metav1.UpdateOptions{})
		}
	}
	{
		list, err := client.KueueV1beta1().MultiKueueClusters().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, item := range list.Items {
			client.KueueV1beta1().MultiKueueClusters().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
			item.Finalizers = nil
			client.KueueV1beta1().MultiKueueClusters().Update(context.Background(), &item, metav1.UpdateOptions{})
		}
	}
	{
		list, err := client.KueueV1beta1().MultiKueueConfigs().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, item := range list.Items {
			client.KueueV1beta1().MultiKueueConfigs().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
			item.Finalizers = nil
			client.KueueV1beta1().MultiKueueConfigs().Update(context.Background(), &item, metav1.UpdateOptions{})
		}
	}
	{
		list, err := client.KueueV1beta1().ProvisioningRequestConfigs().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, item := range list.Items {
			client.KueueV1beta1().ProvisioningRequestConfigs().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
			item.Finalizers = nil
			client.KueueV1beta1().ProvisioningRequestConfigs().Update(context.Background(), &item, metav1.UpdateOptions{})
		}
	}
	{
		list, err := client.KueueV1beta1().ResourceFlavors().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, item := range list.Items {
			client.KueueV1beta1().ResourceFlavors().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
			item.Finalizers = nil
			client.KueueV1beta1().ResourceFlavors().Update(context.Background(), &item, metav1.UpdateOptions{})
		}
	}
	{
		list, err := client.KueueV1beta1().WorkloadPriorityClasses().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, item := range list.Items {
			client.KueueV1beta1().WorkloadPriorityClasses().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
			item.Finalizers = nil
			client.KueueV1beta1().WorkloadPriorityClasses().Update(context.Background(), &item, metav1.UpdateOptions{})
		}
	}
}
