package main

import (
	"fmt"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
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

	// 	get informer
	// factory := informers.NewSharedInformerFactory(clientSet, 10*time.Second)
	factory := informers.NewSharedInformerFactoryWithOptions(clientSet, 10*time.Second, informers.WithNamespace("default"))
	podInformer := factory.Core().V1().Pods().Informer()

	// add work queue
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "controller")
	// 	add event handler
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("add")
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				fmt.Println("can't get key")
			}
			queue.AddRateLimited(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			fmt.Println("update")
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				fmt.Println("can't get key")
			}
			queue.AddRateLimited(key)
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Println("delete")
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				fmt.Println("can't get key")
			}
			queue.AddRateLimited(key)
		},
	})

	// 	 start informer
	stopChan := make(chan struct{})
	factory.Start(stopChan)
	factory.WaitForCacheSync(stopChan)

	<-stopChan
}
