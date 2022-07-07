package pkg

import (
	"context"
	"reflect"
	"time"

	v17 "k8s.io/api/core/v1"
	v15 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v16 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	v13 "k8s.io/client-go/informers/core/v1"
	v14 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	v12 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	workerNum = 5
	maxRetry  = 10
)

type Controller struct {
	client        kubernetes.Interface
	serviceLister v1.ServiceLister
	ingressLister v12.IngressLister
	queue         workqueue.RateLimitingInterface
}

func (c *Controller) Run(stopChan chan struct{}) {
	klog.Info("starting controller")

	for i := 0; i < workerNum; i++ {
		go wait.Until(c.worker, time.Minute, stopChan)
	}

	<-stopChan
}

func (c *Controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c *Controller) updateService(oldObj interface{}, newObj interface{}) {
	// todo compare annotations
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	c.queue.Add(key)
}

func (c *Controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v15.Ingress)
	// get OwnerReference
	ownerReference := v16.GetControllerOf(ingress)

	if ownerReference == nil || ownerReference.Kind != "Service" {
		return
	}

	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
}

func (c *Controller) worker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	defer c.queue.Done(item)

	key := item.(string)
	err := c.syncService(key)
	if err != nil {
		c.handleError(key, err)
	}
	return true
}

func (c *Controller) syncService(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	// 	delete svc
	service, err := c.serviceLister.Services(namespace).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	_, ok := service.GetAnnotations()["ingress/http"]

	ingress, err := c.ingressLister.Ingresses(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if ok && errors.IsNotFound(err) {
		// 	create ingress
		ingressObj := c.constructIngress(service)
		klog.Infof("creating ingress %s in %s", name, namespace)
		_, err := c.client.NetworkingV1().Ingresses(namespace).Create(context.TODO(), ingressObj, v16.CreateOptions{})
		if err != nil {
			klog.Errorf("failed to create ingress %s in %s", name, namespace)
			return err
		}
	} else if !ok && ingress != nil {
		// delete ingress
		err := c.client.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), name, v16.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) handleError(key string, err error) {
	if c.queue.NumRequeues(key) <= maxRetry {
		c.queue.AddRateLimited(key)
		return
	}

	runtime.HandleError(err)
	klog.Infof("forget key %s", key)
	c.queue.Forget(key)
}

func (c *Controller) constructIngress(service *v17.Service) *v15.Ingress {
	pathType := v15.PathTypePrefix
	ingress := v15.Ingress{
		ObjectMeta: v16.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
			OwnerReferences: []v16.OwnerReference{
				*v16.NewControllerRef(service, v17.SchemeGroupVersion.WithKind("Service")),
			},
		},
		Spec: v15.IngressSpec{
			Rules: []v15.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: v15.IngressRuleValue{
						HTTP: &v15.HTTPIngressRuleValue{
							Paths: []v15.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: v15.IngressBackend{
										Service: &v15.IngressServiceBackend{
											Name: service.Name,
											Port: v15.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &ingress
}

func NewController(client kubernetes.Interface, serviceInformer v13.ServiceInformer, ingressInformer v14.IngressInformer) Controller {

	c := Controller{
		client:        client,
		serviceLister: serviceInformer.Lister(),
		ingressLister: ingressInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingress-manager"),
	}

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})

	return c
}
