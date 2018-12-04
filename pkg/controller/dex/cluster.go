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
	"github.com/golang/glog"
	"github.com/kubernetes/kubernetes/cmd/kubeadm/app/util/apiclient"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"

	kubicv1beta1 "github.com/kubic-project/dex-operator/pkg/apis/kubic/v1beta1"
	kubicclient "github.com/kubic-project/dex-operator/pkg/client"
	dexcfg "github.com/kubic-project/dex-operator/pkg/config"
)

const (
	// The namespace where Dex will be run
	dexDefaultNamespace = metav1.NamespaceSystem

	// name of the Dex service
	dexServiceName = "kubic-dex"

	// dexServiceAccountName describes the name of the ServiceAccount for the dex addon
	dexServiceAccountName = "kubic-dex"

	// the network policy name
	dexNetworkPolicyName = "kubic-dex-networkpolicy"

	dexRoleName = "kubic:dex:read-service"

	// dexClusterRoleName sets the name for the dex ClusterRole
	dexClusterRoleName = "kubic:dex"

	dexClusterRoleNameRead = "kubic:dex:read-service"

	dexClusterRoleNameLDAP = "kubic:dex:ldap-administrators"

	// DexClusterRoleNamePSP = "kubic-psp-dex"

	// TODO: maybe configurable in "kubic-init.yaml"...
	dexLDAPAdminGroupName = "Administrators"

	// The image to use for Dex
	dexDefaultImage = "registry.opensuse.org/devel/caasp/kubic-container/container/kubic/caasp-dex:2.7.1"
)

var (
	// https://github.com/kubic-project/salt/blob/master/salt/addons/dex/manifests/05-serviceaccount.yaml
	dexServiceAccount = corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dexServiceAccountName,
			Namespace: dexDefaultNamespace,
			Labels: map[string]string{
				"kubernetes.io/cluster-service": "true",
			},
		},
	}

	dexRoles = []rbac.Role{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dexRoleName,
				Namespace: dexDefaultNamespace,
			},
			Rules: []rbac.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"services"},
					ResourceNames: []string{dexServiceName},
					Verbs:         []string{"get"},
				},
			},
		},
	}

	// https://github.com/kubic-project/salt/blob/master/salt/addons/dex/manifests/05-clusterrole.yaml
	dexClusterRoles = []rbac.ClusterRole{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: dexClusterRoleName,
			},
			Rules: []rbac.PolicyRule{
				{
					APIGroups: []string{"dex.coreos.com"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"create"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dexClusterRoleNameRead,
				Namespace: dexDefaultNamespace,
			},
			Rules: []rbac.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"services"},
					ResourceNames: []string{dexServiceAccountName},
					Verbs:         []string{"get"},
				},
			},
		},
	}

	// https://github.com/kubic-project/salt/blob/master/salt/addons/dex/manifests/10-clusterrolebinding.yaml
	dexClusterRoleBindings = []rbac.ClusterRoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: dexClusterRoleName,
			},
			RoleRef: rbac.RoleRef{
				APIGroup: rbac.GroupName,
				Kind:     "ClusterRole",
				Name:     dexClusterRoleName,
			},
			Subjects: []rbac.Subject{
				{
					Kind:      rbac.ServiceAccountKind,
					Name:      dexServiceAccountName,
					Namespace: dexDefaultNamespace,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: dexClusterRoleNameLDAP,
			},
			RoleRef: rbac.RoleRef{
				APIGroup: rbac.GroupName,
				Kind:     "ClusterRole",
				Name:     dexcfg.DefaultClusterAdminRole,
			},
			Subjects: []rbac.Subject{
				{
					Kind:      rbac.GroupKind,
					Name:      "ADMIN", // To be set...
					Namespace: dexDefaultNamespace,
				},
			},
		},
		// 	For PSP: {
		// 	ObjectMeta: metav1.ObjectMeta{
		// 		Name: DexClusterRoleNamePSP,
		// 	},
		// 	RoleRef: rbac.RoleRef{
		// 		APIGroup: rbac.GroupName,
		// 		Kind:     "ClusterRole",
		// 		Name:     "suse:kubic:psp:privileged",
		// 	},
		// 	Subjects: []rbac.Subject{
		// 		{
		// 			Kind:      rbac.ServiceAccountKind,
		// 			Name:      dexServiceAccountName,
		// 			Namespace: dexDefaultNamespace,
		// 		},
		// 	},
		// }
	}

	// https://github.com/kubic-project/salt/blob/master/salt/addons/dex/manifests/10-rolebinding.yaml
	dexRoleBindings = []rbac.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dexClusterRoleName,
				Namespace: dexDefaultNamespace,
			},
			Subjects: []rbac.Subject{
				{
					Kind:     rbac.GroupKind,
					Name:     "system:authenticated",
					APIGroup: "rbac.authorization.k8s.io",
				},
				{
					Kind:     rbac.GroupKind,
					Name:     "system:unauthenticated",
					APIGroup: "rbac.authorization.k8s.io",
				},
			},
			RoleRef: rbac.RoleRef{
				Kind:     "Role",
				Name:     dexClusterRoleNameRead,
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}

	port389  = intstr.FromInt(389)
	port6444 = intstr.FromInt(6444)
	port53   = intstr.FromInt(53)
	protoTCP = corev1.ProtocolTCP
	protoUDP = corev1.ProtocolUDP

	dexNetworkPolicy = netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dexNetworkPolicyName,
			Namespace: dexDefaultNamespace,
		},
		Spec: netv1.NetworkPolicySpec{
			Egress: []netv1.NetworkPolicyEgressRule{
				{
					Ports: []netv1.NetworkPolicyPort{
						{
							Port:     &port389,
							Protocol: &protoTCP,
						},
						{
							Port:     &port6444,
							Protocol: &protoTCP,
						},
						{
							Port:     &port53,
							Protocol: &protoTCP,
						},
						{
							Port:     &port53,
							Protocol: &protoUDP,
						},
					},
				},
			},
		},
	}

	dexService = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dexServiceName,
			Namespace: dexDefaultNamespace,
			Labels: map[string]string{
				"kubernetes.io/cluster-service": "true",
				"kubernetes.io/name":            "Dex",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{{
				Name:       dexServiceName,
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(5556),
				TargetPort: intstr.FromString("https"),
				NodePort:   0, // To be set...
			},
			},
		},
	}
)

// createOrUpdateDexServiceAccount creates the necessary serviceaccounts that kubeadm uses/might use, if they don't already exist.
func createOrUpdateDexServiceAccount(cli clientset.Interface) error {
	glog.V(3).Infof("[kubic] creating serviceAccount '%s'", dexServiceAccountName)
	if err := apiclient.CreateOrUpdateServiceAccount(cli, &dexServiceAccount); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// deleteDexServiceAccount deletes the ServiceAccount created.
// Note well that it will not fail if the ServiceAccount did not exist.
func deleteDexServiceAccount(cli clientset.Interface) error {
	glog.V(3).Infof("[kubic] deleting serviceAccount '%s'", dexServiceAccountName)

	foregroundDelete := metav1.DeletePropagationForeground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &foregroundDelete,
	}
	err := cli.CoreV1().ServiceAccounts(dexServiceAccount.GetNamespace()).Delete(dexServiceAccount.GetName(), deleteOptions)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// createorUpdateDexRBACRules creates the essential RBAC rules for a minimally set-up cluster
func createorUpdateDexRBACRules(cli clientset.Interface, dexcfg *kubicv1beta1.DexConfiguration) error {
	glog.V(3).Infof("[kubic] creating RBAC rules for Dex")

	cliREST := cli.Discovery().RESTClient()

	for _, cr := range dexClusterRoles {
		glog.V(3).Infof("[kubic] creating ClusterRole '%s'", cr.GetName())
		if err := apiclient.CreateOrUpdateClusterRole(cli, &cr); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
		if err := kubicclient.WaitForObject(cliREST, &cr); err != nil {
			return err
		}
	}

	for _, r := range dexRoles {
		glog.V(3).Infof("[kubic] creating Role '%s'", r.GetName())
		if err := apiclient.CreateOrUpdateRole(cli, &r); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
		if err := kubicclient.WaitForObject(cliREST, &r); err != nil {
			return err
		}
	}

	for _, crb := range dexClusterRoleBindings {
		glog.V(3).Infof("[kubic] creating ClusterRoleBindings '%s'", crb.GetName())

		// set the ADMIN group in the ClusterRoleBindings by detecting the "ADMIN" name
		if crb.Subjects[0].Name == "ADMIN" {
			if len(dexcfg.Spec.AdminGroup) > 0 {
				glog.V(3).Infof("[kubic] using '%s' as Admin group", dexcfg.Spec.AdminGroup)
				crb.Subjects[0].Name = dexcfg.Spec.AdminGroup
			} else {
				crb.Subjects[0].Name = dexLDAPAdminGroupName
			}
		}

		if err := apiclient.CreateOrUpdateClusterRoleBinding(cli, &crb); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
		if err := kubicclient.WaitForObject(cliREST, &crb); err != nil {
			return err
		}
	}

	for _, rb := range dexRoleBindings {
		glog.V(3).Infof("[kubic] creating RoleBindings '%s'", rb.GetName())

		if err := apiclient.CreateOrUpdateRoleBinding(cli, &rb); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
		if err := kubicclient.WaitForObject(cliREST, &rb); err != nil {
			return err
		}
	}

	return nil
}

// deleteDexRBACRules deletes all the RBAC rules created.
// Note well that it will not fail if they did not exist.
// Deletion is performed in foreground mode; i.e. it blocks until/makes sure
// all the resources are deleted.
func deleteDexRBACRules(cli clientset.Interface) error {
	glog.V(3).Infoln("[kubic] deleting RBAC rules for Dex")

	foregroundDelete := metav1.DeletePropagationForeground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &foregroundDelete,
	}

	for _, crb := range dexClusterRoleBindings {
		glog.V(3).Infof("[kubic] deleting ClusterRoleBinding %s", crb.GetName())
		if err := cli.RbacV1().ClusterRoleBindings().Delete(crb.GetName(), deleteOptions); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		// TODO: we should wait for the object to disappear...
	}

	for _, rb := range dexRoleBindings {
		glog.V(3).Infof("[kubic] deleting RoleBinding %s", rb.GetName())
		if err := cli.RbacV1().RoleBindings(rb.GetNamespace()).Delete(rb.GetName(), deleteOptions); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		// TODO: we should wait for the object to disappear...
	}

	for _, cr := range dexClusterRoles {
		glog.V(3).Infof("[kubic] deleting ClusterRole '%s'", cr.GetName())
		if err := cli.RbacV1().ClusterRoles().Delete(cr.GetName(), deleteOptions); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		// TODO: we should wait for the object to disappear...
	}

	for _, r := range dexRoles {
		glog.V(3).Infof("[kubic] deleting Role '%s'", r.GetName())
		if err := cli.RbacV1().Roles(r.GetNamespace()).Delete(r.GetName(), deleteOptions); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		// TODO: we should wait for the object to disappear...
	}

	return nil
}

func createOrUpdateDexService(cli clientset.Interface, dexDeployName string, nodeport int) error {
	// try to replicate the old behaviour in
	// https://github.com/kubic-project/salt/blob/master/salt/addons/dex/manifests/30-network-policy.yaml

	cliREST := cli.Discovery().RESTClient()

	service := dexService.DeepCopy()
	service.Spec.Ports[0].NodePort = int32(nodeport)
	service.Spec.Selector = map[string]string{
		"app": dexDeployName,
	}

	glog.V(3).Infof("[kubic] creating Service '%s' (nodeport=%d)",
		service.GetName(), service.Spec.Ports[0].NodePort)
	if _, err := kubicclient.CreateOrUpdateService(cli, service); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	} else if err := kubicclient.WaitForObject(cliREST, service); err != nil {
		return err
	}
	return nil
}

func deleteDexService(cli clientset.Interface) error {
	glog.V(3).Infof("[kubic] removing Service '%s'", dexService.GetName())
	if err := kubicclient.DeleteServiceForeground(cli, &dexService); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func createOrUpdateDexNetworkPolicy(cli clientset.Interface, dexDeployName string) error {
	// try to replicate the old behaviour in
	// https://github.com/kubic-project/salt/blob/master/salt/addons/dex/manifests/30-network-policy.yaml

	cliREST := cli.Discovery().RESTClient()

	dexNetworkPolicy.Spec.PodSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": dexDeployName,
		},
	}

	glog.V(3).Infof("[kubic] creating NetworkPolicy '%s'", dexNetworkPolicy.GetName())
	if _, err := kubicclient.CreateOrUpdateNetworkPolicy(cli, &dexNetworkPolicy); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	} else if err := kubicclient.WaitForObject(cliREST, &dexNetworkPolicy); err != nil {
		return err
	}
	return nil
}

func deleteNetworkPolicy(cli clientset.Interface) error {
	glog.V(3).Infof("[kubic] removing NetworkPolicy '%s'", dexNetworkPolicy.GetName())
	if err := kubicclient.DeleteNetworkPolicyForeground(cli, &dexNetworkPolicy); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
