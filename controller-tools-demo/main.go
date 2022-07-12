package main

import (
	"context"
	"fmt"

	v1 "controller-tools-demo/pkg/apis/mj.learn/v1"

	"k8s.io/klog/v2"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		klog.Fatal(err)
	}

	config.APIPath = "/apis/"
	config.NegotiatedSerializer = v1.Codecs.WithoutConversion()
	config.GroupVersion = &v1.GroupVersion

	client, err := rest.RESTClientFor(config)
	if err != nil {
		klog.Fatal(err)
	}

	foo := &v1.Foo{}
	err = client.Get().Namespace("default").Resource("foos").Name("crd-test").Do(context.TODO()).Into(foo)
	if err != nil {
		klog.Fatal(err)
	}

	newObj := foo.DeepCopy()
	newObj.Spec.Name = "test2"

	fmt.Println(foo.Spec.Name)
	fmt.Println(foo.Spec.Replicas)

	fmt.Println(newObj.Spec.Name)
}
