package main

import (
	"context"
	"fmt"

	"code-generator-demo/pkg/generated/clientset/versioned"
	"code-generator-demo/pkg/generated/informers/externalversions"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		klog.Fatalf("failed to build kube config: %s", err.Error())
	}

	clientSet, err := versioned.NewForConfig(config)
	if err != nil {
		klog.Fatalf("failed to build kubernetes client: %s", err.Error())
	}

	items, err := clientSet.CrdV1().Foos("default").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		klog.Fatalf("failed to list foo objects: %s", err.Error())
	}

	for _, item := range items.Items {
		fmt.Println(item.Name)
	}

	factory := externalversions.NewSharedInformerFactory(clientSet, 0)
	fooInformer := factory.Crd().V1().Foos()
	fooInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})
}
