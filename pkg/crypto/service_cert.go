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

package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes/kubernetes/cmd/kubeadm/app/util/apiclient"
	certsv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	certutil "k8s.io/client-go/util/cert"

	"github.com/kubic-project/dex-operator/pkg/util"
)

var defaultCertificateUsages = []certsv1beta1.KeyUsage{
	"digital signature",
	"key encipherment",
	"server auth",
	"client auth",
}

// AutoCert is a certificate for a service that is automatically signed by the Kubernetes CA.
type AutoCert struct {
	// Alternative IPs in the certificate
	IPs []net.IP

	// Alternative names (SANs) in the certificate
	Names []string

	// The Name of the secret where the certificate will be saved
	SecretName string

	// ... with namespace
	SecretNamespace string

	// current v1.Secret
	current *corev1.Secret
}

// NewAutoCert creates a new automatically-signed service certificate
func NewAutoCert(ips []net.IP, names []string, name, namespace string) (*AutoCert, error) {
	if len(namespace) == 0 {
		namespace = metav1.NamespaceSystem
	}

	return &AutoCert{
		IPs:             ips,
		Names:           names,
		SecretName:      name,
		SecretNamespace: namespace,
		current:         nil,
	}, nil
}

// NewServiceCertFromReference creates a new automatically-signed service certificate
func NewServiceCertFromReference(ref corev1.SecretReference) (*AutoCert, error) {
	return &AutoCert{
		SecretName:      ref.Name,
		SecretNamespace: ref.Namespace,
		current:         nil,
	}, nil
}

// GetName returns the AutoCert.SecretName
func (ac AutoCert) GetName() string {
	return ac.SecretName
}

// GetNamespace returns the AutoCert.SecretNamespace
func (ac AutoCert) GetNamespace() string {
	return ac.SecretNamespace
}

// Delete removes the certificate. The certificate doesn't need to have been get/requested.
func (ac *AutoCert) Delete(cli clientset.Interface) error {
	var err error

	err = cli.Core().Secrets(util.NamaspacedObjToMeta(ac).Namespace).Delete(util.NamaspacedObjToMeta(ac).Name, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// GetOrRequest gets a certificate from the secret, or perform a new certificate request
func (ac *AutoCert) GetOrRequest(cli clientset.Interface) (*corev1.Secret, error) {
	var err error

	// use the already obtained copy if we have it
	if ac.current != nil {
		return ac.current, nil
	}

	// ... otherwise, try to get it from the apiserver
	ac.current, err = cli.Core().Secrets(util.NamaspacedObjToMeta(ac).Namespace).Get(util.NamaspacedObjToMeta(ac).Name, metav1.GetOptions{})
	if err == nil {
		glog.V(3).Infof("[kubic] TLS secret %q already present in the apiserver", ac.SecretName)
		// TODO: we should check the certificate is still valid
	} else {
		if apierrors.IsNotFound(err) {
			// ... and, if it is not there, request it from the apiserver
			if ac.current, err = ac.Request(cli); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	glog.V(3).Infof("[kubic] using TLS secret '%q'", ac.current.GetName())
	return ac.current, nil
}

// Refresh invalidates the local cached Secret and performs a new GetOrRequest()
func (ac *AutoCert) Refresh(cli clientset.Interface) (*corev1.Secret, error) {
	ac.current = nil
	return ac.GetOrRequest(cli)
}

// Request sends a CSR to the apiserver, requesting auto-approval and waiting until it is approved
func (ac *AutoCert) Request(cli clientset.Interface) (*corev1.Secret, error) {

	csrName := fmt.Sprintf("%s-csr", ac.SecretName)
	csrNamespace := ac.SecretNamespace

	// Generate a private key, pem encode it
	// The private key will be used to create a certificate signing request (csr)
	// that will be submitted to a Kubernetes CA to obtain a TLS certificate.
	glog.V(3).Infof("[kubic] generating private key for '%s'", csrName)
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, fmt.Errorf("unable to genarate the private key: %s", err)
	}

	glog.V(3).Infof("[kubic] creating a CSR for '%s'", csrName)
	certificateRequestTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: ac.Names[0],
		},
		SignatureAlgorithm: x509.SHA256WithRSA,
		DNSNames:           ac.Names,
		IPAddresses:        ac.IPs,
	}

	certificateRequest, err := x509.CreateCertificateRequest(rand.Reader, &certificateRequestTemplate, key)
	if err != nil {
		return nil, fmt.Errorf("unable to generate the CSR: %s", err)
	}

	certificateRequestBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: certificateRequest})

	// Submit a certificate signing request, wait for it to be approved, then save
	// the signed certificate to the file system.
	certificateSigningRequest := &certsv1beta1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      csrName,
			Namespace: csrNamespace,
		},
		Spec: certsv1beta1.CertificateSigningRequestSpec{
			Groups:  []string{"system:authenticated"},
			Request: certificateRequestBytes,
			Usages:  defaultCertificateUsages,
		},
		Status: certsv1beta1.CertificateSigningRequestStatus{
			Conditions: []certsv1beta1.CertificateSigningRequestCondition{},
		},
	}

	glog.V(3).Infof("[kubic] submitting the CSR for '%s'", csrName)
	certificateSigningRequest, err = cli.Certificates().CertificateSigningRequests().Create(certificateSigningRequest)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			glog.V(3).Infof("[kubic] CSR '%s' already exists... getting current version", csrName)
			certificateSigningRequest, err = cli.Certificates().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unable to create the certificate signing request: %s", err)
		}
	}

	certificateSigningRequest.Status.Conditions = append(certificateSigningRequest.Status.Conditions,
		certsv1beta1.CertificateSigningRequestCondition{
			Type:           certsv1beta1.CertificateApproved,
			Reason:         "AutoApproved",
			Message:        "This CSR was approved by the Kubcic Service certificates generator.",
			LastUpdateTime: metav1.Now(),
		})

	_, err = cli.Certificates().CertificateSigningRequests().UpdateApproval(certificateSigningRequest)
	if err != nil {
		return nil, fmt.Errorf("error updating approval for CSR: %v", err)
	}

	glog.V(3).Infof("[kubic] waiting for '%s' to be accepted and signed...", csrName)
	var certificate []byte
	for {
		csr, err := cli.Certificates().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
		if err != nil {
			glog.V(3).Infof("[kubic] unable to retrieve CSR '%s': %s", csrName, err)
			time.Sleep(5 * time.Second)
			continue
		}

		status := csr.Status
		if len(status.Conditions) > 0 {
			if status.Conditions[0].Type == certsv1beta1.CertificateApproved && len(status.Certificate) > 0 {
				glog.V(3).Infof("[kubic] certificate '%s' is approved and signed now", csrName)
				certificate = status.Certificate

				err := cli.Certificates().CertificateSigningRequests().Delete(csrName, &metav1.DeleteOptions{})
				if err != nil && !apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("error removing CSR: %v", err)
				}

				break
			}
		}

		glog.V(3).Infof("[kubic] certificate signing request '%s' not approved yet; trying again in 5 seconds", csrName)
		time.Sleep(5 * time.Second)
	}

	glog.V(3).Infof("[kubic] certificate '%s' signed; uploading to Secret '%s'",
		csrName, util.NamespacedObjToString(ac))
	secret := &corev1.Secret{
		ObjectMeta: util.NamaspacedObjToMeta(ac),
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       certificate,
			corev1.TLSPrivateKeyKey: certutil.EncodePrivateKeyPEM(key),
		},
	}

	if err = apiclient.CreateOrUpdateSecret(cli, secret); err != nil {
		ac.current = nil
		return nil, err
	}
	ac.current, err = cli.CoreV1().Secrets(secret.GetNamespace()).Get(secret.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret, nil
}
