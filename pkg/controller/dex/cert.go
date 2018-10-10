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
	"crypto/sha256"
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/kubic-project/dex-operator/pkg/crypto"
	"github.com/kubic-project/dex-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubicv1beta1 "github.com/kubic-project/dex-operator/pkg/apis/kubic/v1beta1"
	dexcfg "github.com/kubic-project/dex-operator/pkg/config"
	dexnet "github.com/kubic-project/dex-operator/pkg/net"
)

type DexCertificate struct {
	instance *kubicv1beta1.DexConfiguration

	existing   *corev1.Secret
	generated  *corev1.Secret
	reconciler *ReconcileDexConfiguration
}

func NewDexCertificate(instance *kubicv1beta1.DexConfiguration, reconciler *ReconcileDexConfiguration) (*DexCertificate, error) {
	cert := &DexCertificate{
		instance,
		nil,
		nil,
		reconciler,
	}

	if err := cert.GetFrom(instance); err != nil {
		return nil, err
	}
	return cert, nil
}

// GetFrom obtains the existing cert from the GeneratedCertificate specified in the instance.Status or instance.Spec
func (cert *DexCertificate) GetFrom(instance *kubicv1beta1.DexConfiguration) error {
	var err error
	var name, namespace string

	if len(instance.Spec.Certificate.Name) > 0 {
		name, namespace = instance.Spec.Certificate.Name, instance.Spec.Certificate.Namespace
	} else if len(instance.Status.GeneratedCertificate.Name) > 0 {
		name, namespace = instance.Status.GeneratedCertificate.Name, instance.Status.GeneratedCertificate.Namespace
	} else {
		// try to find the certificate we usually generate
		name, namespace = cert.GetName(), cert.GetNamespace()
	}

	if len(name) > 0 {
		cert.existing, err = cert.reconciler.Clientset.Core().Secrets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			cert.existing = nil
			if !apierrors.IsNotFound(err) {
				return err
			}
		} else {
			glog.V(3).Infof("[kubic] there is an existing certificate for Dex")
		}
		cert.generated = nil
	}

	return nil
}

func (cert DexCertificate) WasGenerated() bool {
	return cert.generated != nil
}

func (cert DexCertificate) GetHashRequested() string {
	var data []byte
	if cert.existing != nil {
		data = cert.existing.Data[corev1.TLSCertKey]
	} else if cert.generated != nil {
		data = cert.generated.Data[corev1.TLSCertKey]
	} else {
		panic("internal error: no generated or existing secret")
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// CreateOrUpdate creates the Service in the apiserver, or updates an existing instance
func (cert *DexCertificate) CreateOrUpdate(deployment *DexDeployment) error {

	// TODO: check if cert.existing is valid
	if cert.existing != nil {
		glog.V(3).Infof("[kubic] Dex's service certificate is already in the apiserver: no need to update/create")
		return nil
	}

	defaultAddress, err := dexnet.GetBindIP()
	if err != nil {
		return err
	}

	certIPs := []net.IP{
		defaultAddress,
		net.ParseIP("127.0.0.1"),
	}
	certNames := []string{
		dexServiceName,
		dexnet.GetServiceDNSName(deployment),
		deployment.GetName(),
	}
	certNames = append(certNames, cert.instance.Spec.Names...)

	certificate, err := crypto.NewAutoCert(certIPs, certNames, cert.GetName(), cert.GetNamespace())
	if err != nil {
		return err
	}
	cert.reconciler.EventRecorder.Event(cert.instance, corev1.EventTypeNormal,
		"Checking", fmt.Sprintf("Getting certificate '%s' for '%s'...", certificate.GetName(), cert.instance.GetName()))
	cert.generated, err = certificate.GetOrRequest(cert.reconciler.Clientset)
	if err != nil {
		glog.V(3).Infof("[kubic] could not create/update certificate '%s': %s", util.NamespacedObjToString(cert), err)
		cert.generated = nil
		return err
	}

	return nil
}

func (cert *DexCertificate) Delete() error {
	if cert.generated != nil {
		err := cert.reconciler.Clientset.Core().Secrets(cert.GetNamespace()).Delete(cert.GetName(), &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (cert *DexCertificate) GetObject() metav1.Object {
	if cert.generated != nil {
		return cert.generated
	}
	return nil
}

func (cert DexCertificate) GetName() string {
	if cert.existing != nil {
		return cert.existing.GetName()
	}
	return fmt.Sprintf("%s-auto-cert", dexcfg.DefaultPrefix)
}

func (cert DexCertificate) GetNamespace() string {
	if cert.existing != nil {
		return cert.existing.GetNamespace()
	}
	return metav1.NamespaceSystem
}

func (cert DexCertificate) String() string {
	return util.NamespacedObjToString(cert)
}

func (cert DexCertificate) AsSecretReference() corev1.SecretReference {
	return corev1.SecretReference{
		cert.GetName(),
		cert.GetNamespace(),
	}
}
