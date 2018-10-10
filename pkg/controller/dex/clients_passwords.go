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
	"fmt"

	"github.com/golang/glog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	dexcfg "github.com/kubic-project/dex-operator/pkg/config"
	"github.com/kubic-project/dex-operator/pkg/crypto"
	"github.com/kubic-project/dex-operator/pkg/util"
)

const (
	sharedPasswordNamespace = metav1.NamespaceSystem
)

// StaticClientsPasswords is a groups of static, shared passwords that can be saved
// to k8s Secrets.
type StaticClientsPasswords struct {
	Passwords map[string]crypto.SharedPassword
	Prefix    string
	Namespace string
}

// NewStaticClientsPasswords creates all the shared passwords
// it tries to load those passwords from Secrets in the apiserver
// if they are not found, new random passwords are generated,
// but not persisted in the apiserver
func NewStaticClientsPasswords(prefix string, namespace string) (StaticClientsPasswords, error) {
	// by default, passwords are stored in the "kube-system" namespace
	if len(namespace) == 0 {
		namespace = sharedPasswordNamespace
	}

	sps := StaticClientsPasswords{
		Prefix:    util.SafeId(prefix),
		Namespace: namespace,
		Passwords: map[string]crypto.SharedPassword{},
	}

	return sps, nil
}

// GetOrRandomFromSecrets tries to get the passwords from Secrets or generate random values
func (scp *StaticClientsPasswords) GetOrRandomFromSecrets(cli clientset.Interface, names []string) error {
	glog.V(8).Infof("[kubic] creating/getting %d shared passwords", len(names))
	for _, n := range names {
		fullName := fmt.Sprintf("%s-%s", scp.Prefix, util.SafeId(n))
		glog.V(8).Infof("[kubic] generating/getting shared password '%s'", fullName)
		sharedPassword := crypto.NewSharedPassword(fullName, scp.Namespace)
		if err := sharedPassword.GetFromSecret(cli); apierrors.IsNotFound(err) {
			glog.V(8).Infof("[kubic] shared password '%s' not found: generating random value", fullName)
			if _, err := sharedPassword.Rand(dexcfg.DefaultSharedPasswordLen); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		scp.Passwords[n] = sharedPassword
	}
	return nil
}

// CreateOrUpdateToSecrets publishes all the shared passwords as Secrets in the apiserver
func (scp StaticClientsPasswords) CreateOrUpdateToSecrets(cli clientset.Interface) error {
	glog.V(8).Infof("[kubic] publishing %d shared passwords as Secrets", len(scp.Passwords))
	for _, sharedPassword := range scp.Passwords {
		if err := sharedPassword.CreateOrUpdateToSecret(cli); err != nil {
			return err
		}
	}
	return nil
}
