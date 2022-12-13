/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"kubebuilder-demo/controllers/utils"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ingressv1beta1 "kubebuilder-demo/api/v1beta1"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.mj.learn,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.mj.learn,resources=apps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ingress.mj.learn,resources=apps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the App object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	app := &ingressv1beta1.App{}
	// 从缓存中获取 app
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 根据 app 的配置进行处理
	// 1. Deployment 的处理
	deployment := utils.NewDeployment(app)
	if err := controllerutil.SetControllerReference(app, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	// 查找同名 deployment
	d := &v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, d); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, deployment); err != nil {
				r.Recorder.Event(app, "Error", "Create", "创建 Deployment 失败")
				logger.Error(err, "create deploy failed")
				return ctrl.Result{}, err
			}
			app.Status.DeploymentName = d.Name
			r.Recorder.Event(app, corev1.EventTypeNormal, "Create", "创建 Deployment 成功")
		}
	} else {
		if err := r.Update(ctx, deployment); err != nil {
			r.Recorder.Event(app, "Error", "Update", "更新 Deployment 失败")
			return ctrl.Result{}, err
		}
		r.Recorder.Event(app, corev1.EventTypeNormal, "Update", "更新 Deployment 成功")
	}

	// 2. Service的处理
	service := utils.NewService(app)
	if err := controllerutil.SetControllerReference(app, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	// 查找指定 service
	s := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, s); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnableService {
			if err := r.Create(ctx, service); err != nil {
				logger.Error(err, "create service failed")
				return ctrl.Result{}, err
			}
			app.Status.ServiceName = s.Name
		}
		if !errors.IsNotFound(err) && app.Spec.EnableService {
			return ctrl.Result{}, err
		}
	} else {
		if app.Spec.EnableService {
			logger.Info("already existed, skip update service")
		} else {
			if err := r.Delete(ctx, s); err != nil {
				return ctrl.Result{}, err
			}
			app.Status.ServiceName = ""
		}
	}

	// 3. Ingress 的处理,ingress 配置可能为空
	ingress := utils.NewIngress(app)
	if err := controllerutil.SetControllerReference(app, ingress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	i := &netv1.Ingress{}
	if err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, i); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnableIngress {
			if err := r.Create(ctx, ingress); err != nil {
				logger.Error(err, "create ingress failed")
				return ctrl.Result{}, err
			}
			app.Status.IngressName = i.Name
		}
		if !errors.IsNotFound(err) && app.Spec.EnableIngress {
			return ctrl.Result{}, err
		}
	} else {
		if app.Spec.EnableIngress {
			logger.Info("skip update ingress")
		} else {
			if err := r.Delete(ctx, i); err != nil {
				return ctrl.Result{}, err
			}
			app.Status.IngressName = ""
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1beta1.App{}).
		Owns(&v1.Deployment{}).Owns(&netv1.Ingress{}).Owns(&corev1.Service{}).
		Complete(r)
}
