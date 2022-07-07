package main

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		klog.Fatal(err)
		return
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		klog.Fatal(err)
		return
	}

	APIGroup, APIResourceListSlice, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		klog.Fatal(err)
		return
	}

	fmt.Println(APIGroup, APIResourceListSlice)

	for _, singleAPIResourceList := range APIResourceListSlice {

		// GroupVersion是个字符串，例如"apps/v1"
		groupVersionStr := singleAPIResourceList.GroupVersion

		// ParseGroupVersion方法将字符串转成数据结构
		gv, err := schema.ParseGroupVersion(groupVersionStr)

		if err != nil {
			klog.Fatal(err)
			return
		}

		fmt.Println("*****************************************************************")
		fmt.Printf("GV string [%v]\nGV struct [%#v]\nresources :\n", groupVersionStr, gv)

		// APIResources字段是个切片，里面是当前GroupVersion下的所有资源
		for _, singleAPIResource := range singleAPIResourceList.APIResources {
			fmt.Printf("%v\n", singleAPIResource.Name)
		}
	}
}
