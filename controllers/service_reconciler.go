package controllers

import (
	"context"
	"fmt"
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *MeteorReconciler) ReconcileService(name string, ctx *context.Context, req ctrl.Request) error {
	res := &v1.Service{}
	resourceName := r.Meteor.GetName()
	namespacedName := types.NamespacedName{Name: resourceName, Namespace: req.NamespacedName.Namespace}

	logger := log.FromContext(*ctx).WithValues("service", namespacedName)

	newSpec := &v1.ServiceSpec{
		Ports:    []v1.ServicePort{{Name: "http", Protocol: v1.ProtocolTCP, TargetPort: intstr.FromInt(8080), Port: 8080}},
		Selector: r.Meteor.SeedLabels(),
	}

	updateStatus := func(status metav1.ConditionStatus, reason, message string) {
		r.UpdateStatus(r.Meteor, "Service", name, status, reason, message)
	}

	if err := r.Get(*ctx, namespacedName, res); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Creating Service")

			res = &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: req.NamespacedName.Namespace,
				},
				Spec: *newSpec,
			}

			controllerutil.SetControllerReference(r.Meteor, res, r.Scheme)
			if err := r.Create(*ctx, res); err != nil {
				logger.Error(err, "Unable to create Service")
				updateStatus(metav1.ConditionFalse, "Error", fmt.Sprintf("Unable to create service. %s", err))
				return err
			}

			updateStatus(metav1.ConditionTrue, "Created", "Service was created.")
			return nil
		}
		logger.Error(err, "Error fetching Service")
		updateStatus(metav1.ConditionFalse, "Error", fmt.Sprintf("Reconcile resulted in error. %s", err))
		return err
	}

	if !reflect.DeepEqual(res.Spec.Selector, newSpec.Selector) || !reflect.DeepEqual(res.Spec.Ports, newSpec.Ports) {
		res.Spec = *newSpec
		if err := r.Update(*ctx, res); err != nil {
			logger.Error(err, "Unable to update Service")
			updateStatus(metav1.ConditionFalse, "Error", fmt.Sprintf("Unable to update service. %s", err))
			return err
		}
	}
	updateStatus(metav1.ConditionTrue, "Ready", "Service was reconciled successfully.")
	return nil
}
