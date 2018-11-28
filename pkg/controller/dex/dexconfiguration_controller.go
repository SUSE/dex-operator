/*
 * Copyright 2018 SUSE LINUX GmbH, Nuernberg, Germany..
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package dex

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kubicv1beta1 "github.com/kubic-project/dex-operator/pkg/apis/kubic/v1beta1"
	dexcfg "github.com/kubic-project/dex-operator/pkg/config"
)

const (
	// Name of the controller
	dexControllerName = "DexController"

	// Name of the finalizer
	dexFinalizerName = "dexconfiguration.finalizers.kubic.opensuse.org"

	// Dex main configuration name
	dexMainConfigName = "dex-configuration"
)

var (
	dexDefaultStaticClient = kubicv1beta1.DexStaticClient{
		Name:         "kubernetes",
		RedirectURLs: []string{"urn:ietf:wg:oauth:2.0:oob"},
	}
)

// Add creates a new DexConfiguration Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDexConfiguration{
		Clientset:     clientset.NewForConfigOrDie(mgr.GetConfig()),
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetRecorder(dexControllerName),
		scheme:        mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(dexControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to DexConfiguration and LDAPConnectors
	err = c.Watch(&source.Kind{Type: &kubicv1beta1.DexConfiguration{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes in LDAPConnectors
	// They all go to the global DexConfiguration instance
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      dexMainConfigName,
						Namespace: a.Meta.GetNamespace(),
					}},
			}
		})
	err = c.Watch(&source.Kind{Type: &kubicv1beta1.LDAPConnector{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn})
	if err != nil {
		return err
	}

	// Watch Deployments created by DexConfiguration
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubicv1beta1.DexConfiguration{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDexConfiguration{}

// ReconcileDexConfiguration reconciles a DexConfiguration object
type ReconcileDexConfiguration struct {
	client.Client
	Clientset clientset.Interface
	record.EventRecorder
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a DexConfiguration object and makes
// changes based on the state read and what is in the DexConfiguration.Spec
//
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=core,resources=configmaps;secrets;serviceaccounts;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests/approval;certificatesigningrequests/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubic.opensuse.org,resources=dexconfigurations;ldapconnectors,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileDexConfiguration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	var err error

	ctx := context.Background()
	rr := reconcile.Result{}

	// Fetch the DexConfiguration instance
	instance := &kubicv1beta1.DexConfiguration{}
	if err = r.Get(ctx, request.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	glog.V(3).Infof("[kubic] ******* processing DexConfiguration instance '%s' *******", instance.GetName())

	if instance.GetName() != dexMainConfigName {
		msg := fmt.Sprintf("Dex configuration instance '%s' ignored: the only recognized instance is name=%s",
			instance.GetName(), dexMainConfigName)
		glog.V(3).Infoln(msg)
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "Error", msg)
		return reconcile.Result{}, nil
	}

	// We need some shared secrets
	// (these secrets must be in the same namespace)
	staticClientsPasswords, err := NewStaticClientsPasswords(dexcfg.DefaultPrefix, instance.GetNamespace())
	if err != nil {
		return reconcile.Result{}, err
	}

	staticClients := append(instance.Spec.StaticClients, dexDefaultStaticClient)
	staticClientsNames := []string{}
	for _, sc := range staticClients {
		staticClientsNames = append(staticClientsNames, sc.Name)
	}
	if err = staticClientsPasswords.GetOrRandomFromSecrets(r.Clientset, staticClientsNames); err != nil {
		return reconcile.Result{}, err
	}

	// Generate a new config file
	configMap, err := NewDexConfigMapFor(instance, r)
	if err != nil {
		return reconcile.Result{}, err
	}

	deployment, err := NewDeploymentFor(instance, r)
	if err != nil {
		return reconcile.Result{}, err
	}

	// check if the object is being removed and, in this case, delete all related objects
	finalizing, err := r.finalizerCheck(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if finalizing {
		err = r.reconcileRemoval(instance, deployment, configMap, staticClientsPasswords)
		r.finalizerDone(instance)
	} else {
		rr, err = r.reconcileInstance(instance, deployment, configMap, staticClientsPasswords)
	}

	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "Error", fmt.Sprintf("%s", err))
	}

	// update the instance (despite any previous error)
	glog.V(3).Infof("[kubic] updating Status in DexConfiguration instance '%s'", instance.GetName())
	if err := r.Update(ctx, instance); err != nil {
		glog.V(3).Infof("[kubic] ERROR: when updating DexConfiguration instance '%s': %s", instance.GetName(), err)
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
	}
	glog.V(3).Infof("[kubic] .status successfully updated for DexConfiguration '%s'", instance.GetName())

	return rr, err
}

// reconcileInstance reconciles an instance that must be prrsent in the cluster
func (r *ReconcileDexConfiguration) reconcileInstance(instance *kubicv1beta1.DexConfiguration, deployment *Deployment,
	configMap *ConfigMap, staticClientPasswords StaticClientsPasswords) (reconcile.Result, error) {

	var err error

	connectors, err := r.getLDAPConnectors()
	if err != nil {
		return reconcile.Result{}, err
	}

	// If no connectors are available, Dex should not be running at all
	if len(connectors) == 0 && deployment.IsRunning() {
		glog.V(3).Infof("[kubic] no LDAP connectors available, and Dex was running: removing Deployment '%s'", deployment.GetName())
		if err = r.reconcileRemoval(instance, deployment, configMap, staticClientPasswords); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	instance.Status.NumConnectors = len(connectors)

	if err = configMap.CreateLocal(connectors, staticClientPasswords); err != nil {
		glog.V(3).Infof("[kubic] ERROR: when creating Dex ConfigMap: %s", err)
		return reconcile.Result{}, err
	}
	if err = r.setOwner(instance, configMap); err != nil {
		return reconcile.Result{}, err
	}

	if !configMap.NeedsCreateOrUpdate() {
		glog.V(3).Infoln("[kubic] Dex ConfigMap is still valid: nothing to do.")
		return reconcile.Result{}, nil
	}

	glog.V(3).Infoln("[kubic] Dex ConfigMap is missing or has changed: will be created/updated...")
	r.EventRecorder.Event(instance, corev1.EventTypeNormal,
		"Checking", fmt.Sprintf("ConfigMap '%s' for '%s' has changed",
			configMap.GetName(), instance.GetName()))

	// Get a valid certificate, signed by the CA, for Dex
	certificate, err := NewCertificate(instance, r)
	if err != nil {
		glog.V(3).Infof("[kubic] ERROR: when creating Dex certificate: %s", err)
		return reconcile.Result{}, err
	}
	if err := certificate.CreateOrUpdate(deployment); err != nil {
		glog.V(3).Infof("[kubic] ERROR: when creating/updating Dex certificate: %s", err)
		return reconcile.Result{}, err
	}
	if certificate.WasGenerated() {
		instance.Status.GeneratedCertificate = certificate.AsSecretReference()
	}
	if err = r.setOwner(instance, certificate); err != nil {
		return reconcile.Result{}, err
	}

	// Generate the deployment and create/update it
	if err = deployment.CreateLocal(configMap, certificate); err != nil {
		glog.V(3).Infof("[kubic] ERROR: when creating Dex Deployment: %s", err)
		return reconcile.Result{}, err
	}
	if err = r.setOwner(instance, deployment); err != nil {
		return reconcile.Result{}, err
	}

	if !deployment.NeedsCreateOrUpdate() {
		glog.V(3).Infoln("[kubic] Dex Deployment is still valid: nothing to do.")
		return reconcile.Result{}, nil
	}

	glog.V(3).Infoln("[kubic] Dex deployment is missing or has changed: will be created/updated...")
	r.EventRecorder.Event(instance, corev1.EventTypeNormal,
		"Checking", fmt.Sprintf("Deployment '%s' for '%s' has changed",
			deployment.GetName(), instance.GetName()))
	r.EventRecorder.Event(instance, corev1.EventTypeNormal,
		"Deploying", fmt.Sprintf("Starting/updating Dex..."))

	if err := staticClientPasswords.CreateOrUpdateToSecrets(r.Clientset); err != nil {
		return reconcile.Result{}, err
	}
	instance.Status.StaticClients = []kubicv1beta1.DexStaticClientStatus{}
	for name, password := range staticClientPasswords.Passwords {
		instance.Status.StaticClients = append(instance.Status.StaticClients, kubicv1beta1.DexStaticClientStatus{
			Name:     name,
			Password: password.AsSecretReference(),
			// TODO: update the other fields in the Status
		})
	}
	r.EventRecorder.Event(instance, corev1.EventTypeNormal,
		"Deploying", fmt.Sprintf("Created %d Secrets for shared passwords for '%s'",
			len(staticClientPasswords.Passwords), instance.GetName()))

	if err = configMap.CreateOrUpdate(); err != nil {
		return reconcile.Result{}, err
	}
	instance.Status.Config = configMap.String()
	r.EventRecorder.Event(instance, corev1.EventTypeNormal,
		"Deploying", fmt.Sprintf("Configmap '%s' created for '%s'",
			configMap.GetName(), instance.GetName()))

	if err = deployment.CreateOrUpdate(); err != nil {
		return reconcile.Result{}, err
	}
	instance.Status.Deployment = deployment.String()
	r.EventRecorder.Event(instance, corev1.EventTypeNormal,
		"Deploying", fmt.Sprintf("Deployment '%s' created for '%s'",
			deployment.GetName(), instance.GetName()))

	return reconcile.Result{}, nil
}

// reconcileRemoval ensures that all things created by the controller for a DexConfiguration
// are removed from the apiserver.
// Ensure that delete implementation is idempotent and safe to invoke
// multiple types for same object.
func (r *ReconcileDexConfiguration) reconcileRemoval(instance *kubicv1beta1.DexConfiguration, deployment *Deployment,
	configMap *ConfigMap, staticClientsPasswords StaticClientsPasswords) error {

	var err error

	glog.V(5).Infof("[kubic] deleting all the dependencies for %s", instance.GetName())
	r.EventRecorder.Event(instance, corev1.EventTypeNormal,
		"Removing", fmt.Sprintf("Removing all the dependencies for '%s'...", instance.GetName()))

	if len(instance.Status.Deployment) > 0 {
		if err = deployment.Delete(); err != nil {
			// ignore the deletion error
			glog.V(5).Infof("[kubic] ERROR: could not remove Deployment '%s' for '%s': %s", deployment, instance.GetName(), err)
		} else {
			r.EventRecorder.Event(instance, corev1.EventTypeNormal,
				"Removing", fmt.Sprintf("Deployment '%s' removed", deployment.GetName()))
		}
		instance.Status.Deployment = ""
	}

	if len(instance.Status.Config) > 0 {
		if err = configMap.Delete(); err != nil {
			// ignore the deletion error
			glog.V(5).Infof("[kubic] ERROR: could not remove configmap '%s' for %s: %s",
				configMap.GetName(), instance.GetName(), err)
		} else {
			r.EventRecorder.Event(instance, corev1.EventTypeNormal,
				"Removing", fmt.Sprintf("Configmap '%s' removed", configMap.GetName()))
		}
		instance.Status.Config = ""
	}

	// remove the staticClientsPasswords and the certificate
	for _, password := range staticClientsPasswords.Passwords {
		glog.V(5).Infof("[kubic] removing shared password '%s'", password.GetName())
		if err := password.Delete(r.Clientset); err != nil {
			// ignore the deletion error
			glog.V(5).Infof("[kubic] ERROR: removing shared password '%s' for '%s': %s",
				password.GetName(), instance.GetName(), err)
		}
	}
	instance.Status.StaticClients = []kubicv1beta1.DexStaticClientStatus{}

	// remove the certificate we have generated
	if len(instance.Status.GeneratedCertificate.Name) > 0 {
		cert, _ := NewCertificate(instance, r)
		if err := cert.Delete(); err != nil {
			glog.V(5).Infof("[kubic] ERROR: removing certificate '%s' for '%s': %s",
				cert.GetName(), instance.GetName(), err)
		}
		instance.Status.GeneratedCertificate = corev1.SecretReference{}
	}

	instance.Status.NumConnectors = 0

	return nil
}

// ObjectVisitor interface
type ObjectVisitor interface {
	GetObject() metav1.Object
}

func (r *ReconcileDexConfiguration) setOwner(instance metav1.Object, obj ObjectVisitor) error {
	if robj := obj.GetObject(); robj != nil {
		return controllerutil.SetControllerReference(instance, robj, r.scheme)
	}
	return nil
}

// getLDAPCOnnectors gets the list of LDAP connectors
func (r *ReconcileDexConfiguration) getLDAPConnectors() ([]kubicv1beta1.LDAPConnector, error) {
	// Get the list of LDAP connectors
	connectors := &kubicv1beta1.LDAPConnectorList{}
	if err := r.List(context.TODO(), &client.ListOptions{}, connectors); err != nil {
		return nil, err
	}

	// for _, c := range connectors.Items {
	// }

	return connectors.Items, nil
}

// finalizerCheck checks if the object is being finalized and, in that case,
// remove all the related objects
func (r *ReconcileDexConfiguration) finalizerCheck(instance *kubicv1beta1.DexConfiguration) (bool, error) {
	// Helper functions to check and remove string from a slice of strings.
	containsString := func(slice []string, s string) bool {
		for _, item := range slice {
			if item == s {
				return true
			}
		}
		return false
	}

	finalizing := false
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !containsString(instance.ObjectMeta.Finalizers, dexFinalizerName) {
			glog.V(3).Infof("[kubic] '%s' does not have finalizer '%s' registered: adding it", instance.GetName(), dexFinalizerName)
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, dexFinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return false, err
			}
		}
	} else {
		glog.V(3).Infof("[kubic] '%s' is being deleted", instance.GetName())
		finalizing = true
	}

	return finalizing, nil
}

// finalizerDone marks the instance as "we are done with it, you can remove it now"
// Removal of the `instance` is blocked until we run this function, so make sure you don't
// forget about calling it...
func (r *ReconcileDexConfiguration) finalizerDone(instance *kubicv1beta1.DexConfiguration) error {
	// Helper functions to check and remove string from a slice of strings.
	removeString := func(slice []string, s string) (result []string) {
		for _, item := range slice {
			if item == s {
				continue
			}
			result = append(result, item)
		}
		return
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		panic(fmt.Sprintf("logic error: called finalizerDone() on %s when it was not being destroyed", instance.GetName()))
	}

	glog.V(3).Infof("[kubic] we are done with '%s': it can be safely terminated now.", instance.GetName())
	// remove our finalizer from the list and update it.
	instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, dexFinalizerName)

	return nil
}
