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

//go:generate ../../../build/asset.sh --var configMapTemplate --package dex --in configmap.yaml.in --out configmap

package dex

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path"
	"reflect"

	"github.com/golang/glog"
	"github.com/kubernetes/kubernetes/cmd/kubeadm/app/util/apiclient"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	kubicv1beta1 "github.com/kubic-project/dex-operator/pkg/apis/kubic/v1beta1"
	dexcfg "github.com/kubic-project/dex-operator/pkg/config"
	"github.com/kubic-project/dex-operator/pkg/crypto"
	dexnet "github.com/kubic-project/dex-operator/pkg/net"
	"github.com/kubic-project/dex-operator/pkg/util"
)

type DexConfigMap struct {
	instance *kubicv1beta1.DexConfiguration

	FileName string

	current    *corev1.ConfigMap
	generated  *corev1.ConfigMap
	reconciler *ReconcileDexConfiguration
}

func NewDexConfigMapFor(instance *kubicv1beta1.DexConfiguration, reconciler *ReconcileDexConfiguration) (*DexConfigMap, error) {
	cm := &DexConfigMap{
		instance,
		dexcfg.DefaultConfigMapFilename,
		nil,
		nil,
		reconciler,
	}

	if err := cm.GetFrom(instance); err != nil {
		return nil, err
	}
	return cm, nil
}

// GetFrom obtains the current configmap fromm the ConfigMap specified in the instance.Status
func (config *DexConfigMap) GetFrom(instance *kubicv1beta1.DexConfiguration) error {
	var err error
	var name, namespace string

	// Try to the get current ConfigMap from the data in the instance.Status.Deployment
	if len(instance.Status.Config) > 0 {
		nname := util.StringToNamespacedName(instance.Status.Config)
		name, namespace = nname.Name, nname.Namespace
	} else {
		name, namespace = config.GetName(), config.GetNamespace()
	}

	// try to get the current ConfigMap
	config.current, err = config.reconciler.Clientset.Core().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		config.current = nil
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		glog.V(3).Infof("[kubic] there is an existing ConfigMap for Dex")
	}

	return nil
}

// CreateLocal generates a local ConfigMap instance. Note well that this instance is
// not published to the apiserver: users must use `CreateOrUpdate()` for doing that.
func (config *DexConfigMap) CreateLocal(connectors []kubicv1beta1.LDAPConnector,
	staticClientsPasswords StaticClientsPasswords) error {

	var err error
	var dexAddress string

	glog.V(3).Infoln("[kubic] generating local ConfigMap for Dex")
	if len(config.instance.Spec.Names) > 0 {
		dexAddress = config.instance.Spec.Names[0]
	} else {
		dexAddress, err = dexnet.GetPublicAPIAddress()
		if err != nil {
			return err
		}
	}

	dexPort := dexcfg.DefaultNodePort
	if config.instance.Spec.NodePort != 0 {
		dexPort = config.instance.Spec.NodePort
	}

	glog.V(3).Infof("[kubic] Dex issuer: https://%s:%d", dexAddress, dexPort)
	replacements := struct {
		DexConfigMapFilename string
		DexName              string
		DexNamespace         string
		DexAddress           string
		DexPort              int
		DexSharedPasswords   map[string]crypto.SharedPassword
		DexCertsDir          string
		StaticClients        []kubicv1beta1.DexStaticClient
		LDAPConnectors       []kubicv1beta1.LDAPConnector
	}{
		config.FileName,
		config.GetName(),
		config.GetNamespace(),
		dexAddress,
		dexPort,
		staticClientsPasswords.Passwords,
		dexcfg.DefaultCertsDir,
		config.instance.Spec.StaticClients,
		connectors,
	}

	configMapBytes, err := util.ParseTemplate(configMapTemplate, replacements)
	if err != nil {
		return fmt.Errorf("error when parsing Dex configmap template: %v", err)
	}
	glog.V(8).Infof("[kubic] ConfigMap for Dex:\n%s\n", configMapBytes)

	config.generated = &corev1.ConfigMap{}
	if err := kuberuntime.DecodeInto(clientsetscheme.Codecs.UniversalDecoder(), []byte(configMapBytes), config.generated); err != nil {
		glog.V(3).Infof("[kubic] ConfigMap decoding error: %s", err)
		return fmt.Errorf("unable to decode dex configmap %v", err)
	}
	return nil
}

// NeedsCreateOrUpdate returns true if the ConfigMap is not in the cluster or it needs to be updated
// CreateLocal() must have been previously
func (config DexConfigMap) NeedsCreateOrUpdate() bool {
	if config.generated == nil {
		panic("ConfigMap has not been generated")
	}
	if config.current == nil {
		return true
	}
	return !reflect.DeepEqual(config.generated.Data, config.current.Data)
}

// CreateOrUpdate creates the ConfigMap in the apiserver, or updates an existing instance
func (config *DexConfigMap) CreateOrUpdate() error {
	var err error

	if config.generated == nil {
		// this would be an error in our program's logic
		panic("ConfigMap has not been generated")
	}

	// Create the ConfigMap for Dex or update it in case it already exists
	glog.V(3).Infof("[kubic] creating/updating ConfigMap '%s'", util.NamespacedObjToString(config))
	err = apiclient.CreateOrUpdateConfigMap(config.reconciler.Clientset, config.generated)
	if err != nil {
		glog.V(3).Infof("[kubic] could not create/update ConfigMap '%s': %s", util.NamespacedObjToString(config), err)
		return err
	}

	glog.V(5).Infof("[kubic] ConfigMap '%s' successfully created: refreshing local copy.", util.NamespacedObjToString(config))
	config.current, err = config.reconciler.Clientset.Core().ConfigMaps(config.GetNamespace()).Get(config.GetName(), metav1.GetOptions{})
	if err != nil {
		glog.V(3).Infof("[kubic] could not create/update ConfigMap '%s': %s", util.NamespacedObjToString(config), err)
		config.current = nil
		return err
	}

	return nil
}

// GetHashGenerated returns the hash of the generated config map
func (config DexConfigMap) GetHashGenerated() string {
	data := config.generated.BinaryData
	name := path.Base(dexcfg.DefaultConfigMapFilename)

	// calculate the sha256 of the data in the Configmap
	return fmt.Sprintf("%x", sha256.Sum256(data[name]))
}

// Delete removes the current ConfigMap
func (config *DexConfigMap) Delete() error {
	if config.current != nil {
		if err := config.reconciler.Delete(context.TODO(), config.current); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		config.current = nil
	}

	return nil
}

func (config *DexConfigMap) GetObject() metav1.Object {
	if config.generated == nil {
		panic("needs to be generated first")
	}
	return config.generated
}

func (config DexConfigMap) GetName() string {
	return fmt.Sprintf("%s-cm", dexcfg.DefaultPrefix)
}

func (config DexConfigMap) GetNamespace() string {
	return dexDefaultNamespace
}

func (config DexConfigMap) String() string {
	return util.NamespacedObjToString(config)
}
