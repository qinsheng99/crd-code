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
	codev1 "code/api/v1"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// CodeServerReconciler reconciles a CodeServer object
type CodeServerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Event  chan<- CodeEvent
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CodeServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile

//Reconcile
//+kubebuilder:rbac:groups=code.zjm.com,resources=codeservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=code.zjm.com,resources=codeservers/status,verbs=get;update;patch;create
//+kubebuilder:rbac:groups=code.zjm.com,resources=codeservers/finalizers,verbs=update
func (r *CodeServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	rl := r.Log.WithValues("code", req.NamespacedName)
	code := &codev1.CodeServer{}
	if err := r.Client.Get(ctx, req.NamespacedName, code); err != nil {
		rl.Error(err, "get crd source failed")
		return ctrl.Result{Requeue: false}, err
	}

	if r.findStatusType(&code.Status, codev1.ServerErrored) {
		return ctrl.Result{Requeue: false}, fmt.Errorf("server error")
	} else if r.findStatusType(&code.Status, codev1.ServerRecycled) {
		err := r.Delete(code)
		if err != nil {
			rl.Error(err, "delete crd deployment source failed")
			return ctrl.Result{Requeue: false}, err
		}
		var b bool
		b = r.addStateCondition(&code.Status, r.newStateCondition(codev1.ServerInactive, "过期了,资源被删除了", nil))

		if b {
			_ = r.Status().Update(context.TODO(), code)
			rl.Info("delete crd deployment resource success")
		}
		return ctrl.Result{Requeue: false}, nil
	} else {
		create := r.addStateCondition(&code.Status, r.newStateCondition(codev1.ServerCreated, "resource create", nil))

		_, err := r.createDeployment(code)
		if err != nil {
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 20}, err
		}
		ready := r.addStateCondition(&code.Status, r.newStateCondition(codev1.ServerReady, "resource ready", nil))
		bound := r.addStateCondition(&code.Status, r.newStateCondition(codev1.ServerBound, "resource bound user", nil))
		if create || ready || bound {
			up := code.Status
			if err = r.Get(ctx, req.NamespacedName, code); err != nil {
				rl.Error(err, "get crd source failed")
				return ctrl.Result{}, err
			}
			code.Status = up
			err = r.Status().Update(context.TODO(), code)
			if err != nil {
				rl.Error(err, "update crd source failed")
				return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 20}, err
			}
			rl.Info("crd resource update success")
		}
		d := int64(100)
		if *code.Spec.RecycleAfterSeconds > 0 {
			d = *code.Spec.RecycleAfterSeconds
		}

		r.Event <- CodeEvent{
			duration: d,
			resource: req.NamespacedName,
			t:        metav1.Time{Time: time.Now()},
			flag:     *code.Spec.Add,
		}
	}

	return ctrl.Result{Requeue: false}, nil
}

func (r *CodeServerReconciler) createPod(code *codev1.CodeServer) (*corev1.Pod, error) {
	newPod := r.newPod(code)

	oldPod := &corev1.Pod{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: code.Name, Namespace: code.Namespace}, oldPod)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(context.TODO(), newPod)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	if r.needUpdatePod(oldPod, newPod) {
		oldPod.Spec.Containers = newPod.Spec.Containers

		err = r.Client.Update(context.TODO(), oldPod)
		if err != nil {
			return nil, err
		}
	}
	return oldPod, nil
}

func (r *CodeServerReconciler) createService(code *codev1.CodeServer) (*corev1.Service, error) {
	newService := r.newService(code)
	oldService := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: code.GetName(), Namespace: code.GetNamespace()}, oldService)
	if err != nil && errors.IsNotFound(err) {
		err = r.Client.Create(context.TODO(), newService)
		if err != nil {
			return nil, err
		}
	} else {
		if err != nil {
			return nil, err
		}

		if r.needUpdateService(oldService, newService) {
			oldService.Spec = newService.Spec
			err = r.Client.Update(context.TODO(), oldService)
			if err != nil {
				return nil, err
			}
		}
	}

	return oldService, err
}

func (r *CodeServerReconciler) createDeployment(code *codev1.CodeServer) (*appv1.Deployment, error) {
	rl := r.Log.WithName("ceate deployment")
	newDeployment := r.newDeployment(code)
	oldDeployment := &appv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: code.Name, Namespace: code.Namespace}, oldDeployment)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(context.TODO(), newDeployment)
		if err != nil {
			return nil, err
		}
		rl.Info("deployment create success")
		return newDeployment, nil
	}
	if err != nil {
		return nil, err
	}

	if r.needUpdateDeployment(oldDeployment, newDeployment) {
		oldDeployment.Spec.Template.Spec.Containers = newDeployment.Spec.Template.Spec.Containers
		err = r.Client.Update(context.TODO(), oldDeployment)
		if err != nil {
			return nil, err
		}
	}
	return oldDeployment, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CodeServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&codev1.CodeServer{}).Owns(&corev1.Pod{}).Owns(&corev1.Service{}).Owns(&appv1.Deployment{}).
		Complete(r)
}

func (r *CodeServerReconciler) Delete(code *codev1.CodeServer) error {
	app := &appv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: code.Name, Namespace: code.Namespace}, app)
	if err == nil {
		err = r.Client.Delete(context.TODO(), app)
		if err != nil {
			return err
		}
	}

	if errors.IsNotFound(err) {
		return nil
	}
	return err
}
