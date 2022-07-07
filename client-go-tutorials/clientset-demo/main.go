package main

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		klog.Fatal(err)
		return
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
		return
	}
	pods, err := clientSet.CoreV1().Pods("kube-system").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		klog.Fatal(err)
		return
	}

	for _, pod := range pods.Items {
		fmt.Println(pod.Name)
	}

}
