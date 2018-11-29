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

package util

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// The namespace used when no namespace is provided
	defaultNamespace = metav1.NamespaceSystem
)

// NewNamespacedName returns a new NamespacedName type
func NewNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
}

// NamespacedNameToString returns a string containing [Namespace/Name]
func NamespacedNameToString(ns types.NamespacedName) string {
	if len(ns.Namespace) > 0 {
		return fmt.Sprintf("%s/%s", ns.Namespace, ns.Name)
	}
	return ns.Name
}

// StringToNamespacedName parses a Kubernetes resource name as Namespace and Name
func StringToNamespacedName(name string) types.NamespacedName {
	nname := ""
	nnamespace := ""

	res := strings.SplitN(name, "/", 2)
	if len(res) == 2 {
		nname, nnamespace = res[1], res[0]
	} else {
		nname, nnamespace = res[0], defaultNamespace
	}
	return types.NamespacedName{Name: nname, Namespace: nnamespace}
}

// ObjNamespacer interface
type ObjNamespacer interface {
	GetName() string
	GetNamespace() string
}

// NamespacedObjToNamespacedName returns a NamespaceName created from a ObjNamesapcer
func NamespacedObjToNamespacedName(obj ObjNamespacer) types.NamespacedName {
	return types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

// NamaspacedObjToMeta returns a ObjectMeta created from a ObjNamesapcer
func NamaspacedObjToMeta(obj ObjNamespacer) metav1.ObjectMeta {
	ns := NamespacedObjToNamespacedName(obj)
	return metav1.ObjectMeta{
		Name:      ns.Name,
		Namespace: ns.Namespace,
	}
}

// NamespacedObjToString returns a string representation of a ObjNamespacer in the form of [Namespace/Name]
func NamespacedObjToString(obj ObjNamespacer) string {
	if len(obj.GetNamespace()) > 0 {
		return fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())
	}
	return fmt.Sprintf("%s", obj.GetName())
}
