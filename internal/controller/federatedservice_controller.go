// Copyright Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	federationv1alpha1 "github.com/openshift-service-mesh/federation/api/v1alpha1"
)

// FederatedServiceReconciler reconciles a FederatedService object
type FederatedServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=federation.openshift-service-mesh.io,resources=federatedservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=federation.openshift-service-mesh.io,resources=federatedservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=federation.openshift-service-mesh.io,resources=federatedservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FederatedService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *FederatedServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.FromContext(ctx).Info("Reconciling object", "name", req.Name, "namespace", req.Namespace)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FederatedServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&federationv1alpha1.FederatedService{}).
		Complete(r)
}
