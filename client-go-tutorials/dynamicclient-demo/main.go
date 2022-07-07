package main

import (
	"context"
	"fmt"

	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		klog.Fatal(err)
		return
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
		return
	}

	r := schema.GroupVersionResource{
		Version: "v1", Resource: "pods",
	}

	unstructuredPods, err := dynamicClient.Resource(r).Namespace("kube-system").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		klog.Fatal(err)
		return
	}

	pods := &v12.PodList{}

	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPods.UnstructuredContent(), pods); err != nil {
		panic(err)
	}

	for _, pod := range pods.Items {
		fmt.Println(pod.Name)
	}

}
