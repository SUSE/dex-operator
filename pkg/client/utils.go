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
package client

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	pollInterval = 2 * time.Second

	pollTimeout = 5 * time.Minute
)

// note well: for some objects we cannot try to Create() the object and then Update() if it failed
//            with an AlreadyExists(), because Create() could just fail on some validation (for example,
//            the "port is already in use")

// CreateOrUpdatePod creates or updates a Pod object
func CreateOrUpdatePod(client clientset.Interface, service *corev1.Pod) (*corev1.Pod, error) {
	var err error
	var existing *corev1.Pod = nil

	existing, err = client.Core().Pods(service.GetNamespace()).Get(service.GetName(), metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		if existing, err = client.Core().Pods(service.GetNamespace()).Create(service); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if !reflect.DeepEqual(service.Spec, existing.Spec) {
			existing.Spec = service.Spec
			if existing, err = client.Core().Pods(existing.GetNamespace()).Update(existing); err != nil {
				return nil, fmt.Errorf("unable to update Pod: %v", err)
			}
		}
	}

	return existing, nil
}

// CreateOrUpdateJob creates a Job if the target resource doesn't exist. If the resource exists
// already, this function will update the resource instead.
func CreateOrUpdateJob(client clientset.Interface, job *batchv1.Job) (*batchv1.Job, error) {
	var err error
	var existing *batchv1.Job = nil

	existing, err = client.Batch().Jobs(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		if existing, err = client.Batch().Jobs(job.GetNamespace()).Create(job); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if !reflect.DeepEqual(job.Spec, existing.Spec) {
			existing.Spec = job.Spec
			if existing, err = client.Batch().Jobs(existing.GetNamespace()).Update(existing); err != nil {
				return nil, fmt.Errorf("unable to update Job: %v", err)
			}
		}
	}

	return existing, nil
}

// CreateOrUpdateDeployment creates a Deployment if the target resource doesn't exist. If the resource exists already, this function will update the resource instead.
func CreateOrUpdateService(client clientset.Interface, service *corev1.Service) (*corev1.Service, error) {
	var err error
	var existing *corev1.Service = nil

	existing, err = client.Core().Services(service.GetNamespace()).Get(service.GetName(), metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		if existing, err = client.Core().Services(service.GetNamespace()).Create(service); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if !reflect.DeepEqual(service.Spec, existing.Spec) {
			existing.Spec.Type = service.Spec.Type
			existing.Spec.Ports = service.Spec.Ports
			if existing, err = client.Core().Services(existing.GetNamespace()).Update(existing); err != nil {
				return nil, fmt.Errorf("unable to update Service: %v", err)
			}
		}
	}

	return existing, nil
}

// DeleteServiceForeground deletes a Service
// Deletion is performed in foreground mode; i.e. it blocks until/makes sure
// all the resources are deleted.
func DeleteServiceForeground(client clientset.Interface, service *corev1.Service) error {
	foregroundDelete := metav1.DeletePropagationForeground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &foregroundDelete,
	}

	if err := client.Core().Services(service.GetNamespace()).Delete(service.GetName(), deleteOptions); err != nil {
		return err
	}
	return nil
}

func CreateOrUpdateNetworkPolicy(client clientset.Interface, np *netv1.NetworkPolicy) (*netv1.NetworkPolicy, error) {
	var unp *netv1.NetworkPolicy = nil
	var err error

	if unp, err = client.Networking().NetworkPolicies(np.ObjectMeta.Namespace).Create(np); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("unable to create NetworkPolicy: %v", err)
		}

		if unp, err = client.Networking().NetworkPolicies(np.ObjectMeta.Namespace).Update(np); err != nil {
			return nil, fmt.Errorf("unable to update NetworkPolicy: %v", err)
		}
	}
	return unp, nil
}

// DeleteNetworkPolicyForeground deletes a NetworkPolicy
// Deletion is performed in foreground mode; i.e. it blocks until/makes sure
// all the resources are deleted.
func DeleteNetworkPolicyForeground(client clientset.Interface, np *netv1.NetworkPolicy) error {
	foregroundDelete := metav1.DeletePropagationForeground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &foregroundDelete,
	}

	err := client.Networking().NetworkPolicies(np.ObjectMeta.Namespace).Delete(np.GetName(), deleteOptions)
	if err != nil {
		return err
	}
	return nil
}

// WaitForObject waits for an object to be ready in the apiserver
func WaitForObject(cli rest.Interface, obj metav1.Common) error {
	request := cli.Get().AbsPath(obj.GetSelfLink())
	return WaitForURL(request)
}

// WaitForObject waits for an URL to be GET'able
func WaitForURL(request *rest.Request) error {
	glog.V(5).Infof("[kubic] Waiting until endpoint is available...")
	err := wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		res := request.Do()
		err := res.Error()
		if err != nil {
			// RESTClient returns *apierrors.StatusError for any status codes < 200 or > 206
			// and http.Client.Do errors are returned directly.
			if se, ok := err.(*apierrors.StatusError); ok {
				if se.Status().Code == http.StatusNotFound {
					return false, nil
				}
			}
			return false, err
		}

		var statusCode int
		res.StatusCode(&statusCode)
		if statusCode != http.StatusOK {
			return false, fmt.Errorf("invalid status code: %d", statusCode)
		}

		return true, nil
	})

	return err
}
