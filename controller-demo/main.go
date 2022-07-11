package main

import (
	"time"

	"controller-demo/pkg"
	"controller-demo/pkg/signals"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

const workerNum = 5

func main() {
	stopChan := signals.SetupSignalHandler()

	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Fatalf("failed to build kube config: %s", err.Error())
		}
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("failed to build kubernetes client: %s", err.Error())
	}

	factory := informers.NewSharedInformerFactory(clientSet, time.Second*30)

	serviceInformer := factory.Core().V1().Services()
	ingressInformer := factory.Networking().V1().Ingresses()

	controller := pkg.NewController(clientSet, serviceInformer, ingressInformer)

	factory.Start(stopChan)

	if err := controller.Run(workerNum, stopChan); err != nil {
		klog.Fatalf("failed to run controller: %s", err.Error())
	}
}
