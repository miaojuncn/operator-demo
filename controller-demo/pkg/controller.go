package pkg

import (
	"context"
	"fmt"
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

const maxRetry = 10

type Controller struct {
	client        kubernetes.Interface
	serviceLister v1.ServiceLister
	serviceSynced cache.InformerSynced
	ingressLister v12.IngressLister
	ingressSynced cache.InformerSynced
	queue         workqueue.RateLimitingInterface
}

func (c *Controller) Run(workerNum int, stopChan <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.queue.ShuttingDown()

	klog.Info("starting controller")

	klog.Info("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopChan, c.serviceSynced, c.ingressSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < workerNum; i++ {
		go wait.Until(c.worker, time.Minute, stopChan)
	}
	klog.Info("started workers")

	<-stopChan
	klog.Info("shutting down workers")
	return nil
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

func (c *Controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v15.Ingress)
	// get OwnerReference
	ownerReference := v16.GetControllerOf(ingress)

	if ownerReference == nil || ownerReference.Kind != "Service" {
		return
	}

	// c.queue.Add(ingress.Namespace + "/" + ingress.Name)
	c.enqueue(obj)
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.Add(key)
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

	// 移除队列中的元素
	defer c.queue.Done(item)

	key, ok := item.(string)
	if !ok {
		c.queue.Forget(key)
		klog.Errorf("expected string in work queue but got %#v", item)
		return true
	}

	if err := c.syncService(key); err != nil {
		if c.queue.NumRequeues(key) <= maxRetry {
			c.queue.AddRateLimited(key)
			klog.Warningf("error syncing %s: %s, requeue", key, err.Error())
			return true
		}

		c.queue.Forget(key)
		klog.Infof("successfully synced or reached max retry number %s", key)
		return true
	}

	return true
}

func (c *Controller) syncService(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		klog.Errorf("invalid resource key: %s", key)
		return nil
	}

	service, err := c.serviceLister.Services(namespace).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// annotations map[string][string]
	v := service.GetAnnotations()["ingress/http"]

	ingress, err := c.ingressLister.Ingresses(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if v == "true" && errors.IsNotFound(err) {
		// 	create ingress
		ingressObj := c.constructIngress(service)
		klog.Infof("creating ingress %s in %s", name, namespace)
		_, err := c.client.NetworkingV1().Ingresses(namespace).Create(context.TODO(), ingressObj, v16.CreateOptions{})
		if err != nil {
			klog.Errorf("failed to create ingress %s in %s", name, namespace)
			return err
		}
	} else if v != "true" && ingress != nil {
		// delete ingress
		klog.Infof("deleting ingress %s in %s", name, namespace)
		err := c.client.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), name, v16.DeleteOptions{})
		if err != nil {
			klog.Errorf("failed to delete ingress %s in %s", name, namespace)
			return err
		}
	}
	return nil
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
		serviceSynced: serviceInformer.Informer().HasSynced,
		ingressLister: ingressInformer.Lister(),
		ingressSynced: ingressInformer.Informer().HasSynced,
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
