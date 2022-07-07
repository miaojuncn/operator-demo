package main

import (
	"time"

	"operator-demo/controller-demo/pkg"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Fatal("can't get config")
		}
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal("can't create client")
	}

	factory := informers.NewSharedInformerFactory(clientSet, time.Second*30)
	serviceInformer := factory.Core().V1().Services()
	ingressInformer := factory.Networking().V1().Ingresses()

	controller := pkg.NewController(clientSet, serviceInformer, ingressInformer)

	stopChan := make(chan struct{})
	factory.Start(stopChan)
	factory.WaitForCacheSync(stopChan)

	controller.Run(stopChan)
}
