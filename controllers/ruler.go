package controllers

import (
	codev1 "github.com/qinsheng99/crd-code/api/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *CodeServerReconciler) needUpdateService(old, new *corev1.Service) bool {
	return !equality.Semantic.DeepEqual(old.Spec.Ports, new.Spec.Ports) || !equality.Semantic.DeepEqual(old.Spec.Selector, new.Spec.Selector)
}

func (r *CodeServerReconciler) needUpdateDeployment(old, new *appv1.Deployment) bool {
	return !equality.Semantic.DeepEqual(old.Spec.Template.Spec.Containers, new.Spec.Template.Spec.Containers)
}

func (r *CodeServerReconciler) needUpdatePod(old, new *corev1.Pod) bool {
	return !equality.Semantic.DeepEqual(old.Spec.Containers, new.Spec.Containers)
}

func (r *CodeServerReconciler) newService(code *codev1.CodeServer) *corev1.Service {
	newService := new(corev1.Service)
	newService.ObjectMeta = metav1.ObjectMeta{Name: code.GetName(), Namespace: code.GetNamespace()}
	newService.Spec = corev1.ServiceSpec{Selector: map[string]string{"app": code.GetName()}}
	newService.Spec.Ports = append(newService.Spec.Ports, corev1.ServicePort{Port: 8080, Name: "http", Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(8080)})

	_ = controllerutil.SetControllerReference(code, newService, r.Scheme)
	return newService
}

func (r *CodeServerReconciler) newDeployment(code *codev1.CodeServer) *appv1.Deployment {
	newDeployment := new(appv1.Deployment)

	newDeployment.TypeMeta = metav1.TypeMeta{Kind: code.TypeMeta.Kind, APIVersion: code.TypeMeta.APIVersion}

	newDeployment.ObjectMeta = metav1.ObjectMeta{Name: code.GetName(), Namespace: code.GetNamespace()}

	i := int32(1)

	newDeployment.Spec = appv1.DeploymentSpec{
		Replicas: &i,
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": code.ObjectMeta.Name}},
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{ImagePullPolicy: corev1.PullAlways, Name: "test-server", Image: code.Spec.Image, Env: code.Spec.Envs},
				},
			},
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": code.Name}},
		},
	}
	_ = controllerutil.SetControllerReference(code, newDeployment, r.Scheme)
	return newDeployment
}

func (r *CodeServerReconciler) newPod(code *codev1.CodeServer) *corev1.Pod {
	newPod := new(corev1.Pod)

	newPod.TypeMeta = metav1.TypeMeta{Kind: code.TypeMeta.Kind, APIVersion: code.TypeMeta.APIVersion}

	newPod.ObjectMeta = metav1.ObjectMeta{Name: code.ObjectMeta.Name, Namespace: code.ObjectMeta.Namespace}

	newPod.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "test-server",
				Image:           code.Spec.Image,
				Env:             code.Spec.Envs,
				ImagePullPolicy: corev1.PullAlways,
			},
		},
		RestartPolicy: corev1.RestartPolicyNever,
	}

	_ = controllerutil.SetControllerReference(code, newPod, r.Scheme)

	return newPod
}

func (r *CodeServerReconciler) addStateCondition(status *codev1.CodeServerStatus, news codev1.ServerCondition) bool {
	condition := r.findStatus(status, news.Type)
	if condition != nil {
		return false
	}
	if condition != nil {
		news.LastTransitionTime = condition.LastTransitionTime
	}

	status.Conditions = append(status.Conditions, news)
	return true
}

func (r *CodeServerReconciler) findStatus(status *codev1.CodeServerStatus, typ codev1.ServerConditionType) *codev1.ServerCondition {
	for _, v := range status.Conditions {
		if v.Type == typ {
			return &v
		}
	}

	return nil
}

func (r *CodeServerReconciler) newStateCondition(typ codev1.ServerConditionType, reason string, message map[string]string) codev1.ServerCondition {
	return codev1.ServerCondition{
		Type:               typ,
		Status:             corev1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
	}
}

func (r *CodeServerReconciler) findStatusType(status *codev1.CodeServerStatus, typ codev1.ServerConditionType) (flag bool) {
	for _, condition := range status.Conditions {
		if typ == condition.Type && condition.Status == corev1.ConditionTrue {
			flag = true
			break
		}
	}
	return
}
