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

//go:generate ../../../build/asset.sh --var deploymentTemplate --package dex --in deployment.yaml.in --out deployment

package dex

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/kubernetes/kubernetes/cmd/kubeadm/app/util/apiclient"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	kubicv1beta1 "github.com/kubic-project/dex-operator/pkg/apis/kubic/v1beta1"
	dexcfg "github.com/kubic-project/dex-operator/pkg/config"
	"github.com/kubic-project/dex-operator/pkg/util"
)

// Deployment struct
type Deployment struct {
	DexCfg *kubicv1beta1.DexConfiguration

	current    *appsv1.Deployment
	generated  *appsv1.Deployment
	reconciler *ReconcileDexConfiguration
}

// NewDeploymentFor returns a new Deployment struct for the configuration
func NewDeploymentFor(instance *kubicv1beta1.DexConfiguration, reconciler *ReconcileDexConfiguration) (*Deployment, error) {

	deploy := &Deployment{
		instance,
		nil,
		nil,
		reconciler,
	}

	if err := deploy.GetFrom(instance); err != nil {
		return nil, err
	}
	return deploy, nil
}

// GetFrom obtains the current deployment fromm the Deployment specified in the instance.Status
func (deploy *Deployment) GetFrom(instance *kubicv1beta1.DexConfiguration) error {
	var err error
	var name, namespace string

	// Try to the get current Deployment from the data in the instance.Status.Deployment
	if len(instance.Status.Deployment) > 0 {
		nname := util.StringToNamespacedName(instance.Status.Deployment)
		name, namespace = nname.Name, nname.Namespace
	} else {
		name, namespace = deploy.GetName(), deploy.GetNamespace()
	}

	// try to get any current deployment
	deploy.current, err = deploy.reconciler.Clientset.Apps().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		deploy.current = nil
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		glog.V(3).Infof("[kubic] there is an existing Deployment for Dex")
	}

	return nil
}

// CreateLocal generates a local Deployment instance. Note well that this instance is
// not published to the apiserver: users must use `CreateOrUpdate()` for doing that.
func (deploy *Deployment) CreateLocal(configMap *ConfigMap, cert *Certificate) error {
	var err error

	// some checks: deployment cannot access Secrets in different namespaces
	if cert.GetNamespace() != deploy.GetNamespace() {
		panic("secret and deployment namespaces must match")
	}

	glog.V(3).Infoln("[kubic] generating deployment for Dex")

	configMapSha := configMap.GetHashGenerated()
	if len(configMapSha) == 0 {
		panic("could not get the hash for the ConfigMap.")
	}
	glog.V(3).Infof("[kubic] Deployment: configMap with HASH=%s", configMapSha)

	certSha := cert.GetHashRequested()
	if len(certSha) == 0 {
		panic("could not get the cert hash... was it Request()ed?")
	}
	glog.V(3).Infof("[kubic] Deployment: cert with HASH=%s", certSha)

	image := deploy.DexCfg.Spec.Image
	if len(image) == 0 {
		image = dexDefaultImage
	}

	replicas := deploy.DexCfg.Spec.Replicas
	if replicas == 0 {
		replicas = dexcfg.DefaultDeployNumReplicas
	}

	replacements := struct {
		DexImage              string
		DexServiceAccount     string
		DexName               string
		DexNamespace          string
		DexDeploymentReplicas int
		DexCertsSecretName    string
		DexConfigMapName      string
		DexConfigMapSha       string
		DexConfigMapFilename  string
		DexCertSha            string
		DexCertsDir           string
	}{
		image,
		dexServiceAccountName,
		deploy.GetName(),
		deploy.GetNamespace(),
		replicas,
		cert.GetName(),
		configMap.GetName(),
		configMapSha,
		dexcfg.DefaultConfigMapFilename,
		certSha,
		dexcfg.DefaultCertsDir,
	}

	deploymentBytes, err := util.ParseTemplate(deploymentTemplate, replacements)
	if err != nil {
		glog.V(3).Infof("[kubic] error when parsing Dex deployment template: %v", err)
		return fmt.Errorf("error when parsing Dex deployment template: %v", err)
	}
	glog.V(8).Infof("[kubic] Dex deployment:\n%s\n", deploymentBytes)

	deploy.generated = &appsv1.Deployment{}
	if err := kuberuntime.DecodeInto(clientsetscheme.Codecs.UniversalDecoder(), []byte(deploymentBytes), deploy.generated); err != nil {
		glog.V(3).Infof("[kubic] Deployment decoding error: %s", err)
		return fmt.Errorf("unable to decode dex daemonset %v", err)
	}
	return nil
}

// IsRunning returns true if the Deployment is not in the cluster or it needs to be updated
func (deploy *Deployment) IsRunning() bool {
	return deploy.current != nil
}

// NeedsCreateOrUpdate returns true if the Deployment is not in the cluster or it needs to be updated
// CreateLocal() must have been previously
func (deploy Deployment) NeedsCreateOrUpdate() bool {
	if deploy.generated == nil {
		panic("Deployment has not been generated")
	}
	if deploy.current == nil {
		return true
	}
	return !reflect.DeepEqual(deploy.generated.Spec, deploy.current.Spec)
}

// CreateOrUpdate creates or updates the deployment
func (deploy *Deployment) CreateOrUpdate() error {
	var err error

	if deploy.generated == nil {
		// this would be an error in our program's logic
		panic("Deployment has not been generated")
	}

	if err := createOrUpdateDexServiceAccount(deploy.reconciler.Clientset); err != nil {
		glog.V(3).Infof("[kubic] ERROR: could not create/update Service Account for '%s': %s", util.NamespacedObjToString(deploy), err)
		return err
	}

	if err := createorUpdateDexRBACRules(deploy.reconciler.Clientset, deploy.DexCfg); err != nil {
		glog.V(3).Infof("[kubic] ERROR: could not create/update RBAC rules for '%s': %s", util.NamespacedObjToString(deploy), err)
		return err
	}

	// create/update the current deployment
	glog.V(5).Infof("[kubic] creating Deployment %s", deploy)
	err = apiclient.CreateOrUpdateDeployment(deploy.reconciler.Clientset, deploy.generated)
	if err != nil {
		glog.V(3).Infof("[kubic] ERROR: could not create/update Deployment '%s': %s", util.NamespacedObjToString(deploy), err)
		return err
	}

	glog.V(5).Infof("[kubic] Deployment '%s' successfully created: refreshing local copy.", deploy)
	deploy.current, err = deploy.reconciler.Clientset.AppsV1().Deployments(deploy.generated.GetNamespace()).Get(deploy.generated.GetName(), metav1.GetOptions{})
	if err != nil {
		glog.V(3).Infof("[kubic] ERROR: could not create/update Deployment '%s': %s", util.NamespacedObjToString(deploy), err)
		deploy.current = nil
		return err
	}

	// Crete the Service and the Network Policy
	port := dexcfg.DefaultNodePort
	if deploy.DexCfg.Spec.NodePort != 0 {
		port = deploy.DexCfg.Spec.NodePort
	}

	if err := createOrUpdateDexService(deploy.reconciler.Clientset, deploy.GetName(), port); err != nil {
		glog.V(5).Infof("[kubic] could not create/update Service: %s", err)
		return err
	}

	if err := createOrUpdateDexNetworkPolicy(deploy.reconciler.Clientset, deploy.GetName()); err != nil {
		glog.V(5).Infof("[kubic] could not create/update NetworkPolicy: %s", err)
		return err
	}

	return nil
}

// Delete removes the current deployment as well as all the other resources created
// It will ignore IsNotFound errors.
func (deploy *Deployment) Delete() error {
	if deploy.current != nil {
		if err := apiclient.DeleteDeploymentForeground(deploy.reconciler.Clientset, deploy.GetNamespace(), deploy.GetName()); err != nil {
			return err
		}
		deploy.current = nil

		if err := deleteDexRBACRules(deploy.reconciler.Clientset); err != nil {
			glog.V(3).Infof("[kubic] ERROR: could not delete RBAC rules: %s", err)
			return err
		}

		if err := deleteDexServiceAccount(deploy.reconciler.Clientset); err != nil {
			glog.V(3).Infof("[kubic] ERROR: could not delete ServiceAccount: %s", err)
			return err
		}

		if err := deleteDexService(deploy.reconciler.Clientset); err != nil {
			glog.V(3).Infof("[kubic] ERROR: could not delete Service: %s", err)
			return err
		}

		if err := deleteNetworkPolicy(deploy.reconciler.Clientset); err != nil {
			glog.V(3).Infof("[kubic] ERROR: could not delete NetworkPolicy: %s", err)
			return err
		}
	}

	return nil
}

// GetObject returns the generated metav1.Object
func (deploy Deployment) GetObject() metav1.Object {
	if deploy.generated == nil {
		panic("needs to be generated first")
	}
	return deploy.generated
}

// GetName returns the name of the dex config
func (deploy Deployment) GetName() string {
	return fmt.Sprintf("%s-deploy", dexcfg.DefaultPrefix)
}

// GetNamespace returns the default dex Namespace
func (deploy Deployment) GetNamespace() string {
	return dexDefaultNamespace
}

// String returns the Namespace object as a string
func (deploy Deployment) String() string {
	return util.NamespacedObjToString(deploy)
}
