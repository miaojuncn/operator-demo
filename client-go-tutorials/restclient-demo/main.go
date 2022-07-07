package main

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		klog.Fatal(err)
		return
	}

	config.APIPath = "api"
	// config.NegotiatedSerializer = scheme.Codecs
	// config.GroupVersion = &v1.SchemeGroupVersion

	config.ContentConfig = rest.ContentConfig{
		NegotiatedSerializer: scheme.Codecs,
		GroupVersion:         &v1.SchemeGroupVersion,
	}

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		klog.Fatal(err)
		return
	}

	result := &v1.PodList{}

	err = restClient.Get().Namespace("kube-system").Resource("pods").Do(context.TODO()).Into(result)
	if err != nil {
		klog.Fatal(err)
		return
	}

	for _, pod := range result.Items {
		fmt.Println(pod.Name)
	}
}
