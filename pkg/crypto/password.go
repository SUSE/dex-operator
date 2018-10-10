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
	"math/rand"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes/kubernetes/cmd/kubeadm/app/util/apiclient"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/kubic-project/dex-operator/pkg/util"
)

var (
	sharedPasswordNamespace = metav1.NamespaceSystem

	// the length (in bytes) for these passwords
	sharedPasswordDefaultLen = 16
)

type SharedPassword struct {
	Name     string // The name includes the "namespace" (ie, "kube-system/dex-velum")
	length   int
	contents string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randStringRunes(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func NewSharedPassword(name, namespace string) SharedPassword {
	if len(namespace) == 0 {
		namespace = sharedPasswordNamespace
	}
	return SharedPassword{
		Name: util.NamespacedNameToString(util.NewNamespacedName(name, namespace)),
	}
}

func (password *SharedPassword) Rand(length int) (string, error) {
	if length == 0 {
		length = sharedPasswordDefaultLen
	}
	password.contents = randStringRunes(length)
	return password.contents, nil
}

func (password SharedPassword) GetName() string {
	return util.StringToNamespacedName(password.Name).Name
}

func (password SharedPassword) GetNamespace() string {
	return util.StringToNamespacedName(password.Name).Namespace
}

// String implements the Stringer interface
func (password SharedPassword) String() string {
	return string(password.contents[:])
}

// CreateOrUpdateToSecret publishes a password as a secret
func (password SharedPassword) CreateOrUpdateToSecret(cli clientset.Interface) error {
	secret := &corev1.Secret{
		ObjectMeta: util.NamaspacedObjToMeta(password),
		Type:       corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			password.GetName(): []byte(password.contents),
		},
	}
	if err := apiclient.CreateOrUpdateSecret(cli, secret); err != nil {
		return err
	}
	glog.V(3).Infof("[kubic] created Secret %s for password", password.GetName())
	return nil
}

// GetFromSecret gets the shared password from a Secret
func (password *SharedPassword) GetFromSecret(cli clientset.Interface) error {
	found, err := cli.CoreV1().Secrets(password.GetNamespace()).Get(password.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	glog.V(3).Infof("[kubic] there is an existing password for '%s'", password.GetName())
	password.contents = string(found.Data[password.GetName()])
	return nil
}

func (password *SharedPassword) AsSecretReference() corev1.SecretReference {
	return corev1.SecretReference{
		Name:      password.GetName(),
		Namespace: password.GetNamespace(),
	}
}

func (password *SharedPassword) Delete(cli clientset.Interface) error {
	err := cli.CoreV1().Secrets(password.GetNamespace()).Delete(password.GetName(), &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
