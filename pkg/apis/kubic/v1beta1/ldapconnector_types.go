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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// see https://github.com/dexidp/dex/blob/master/Documentation/connectors/ldap.md

// User search maps a username and password entered by a user to a LDAP entry.
type LDAPUserSpec struct {
	// BaseDN to start the search from. It will translate to the query
	// "(&(objectClass=person)(uid=<username>))".
	BaseDN string `json:"baseDn,omitempty"`

	// Optional filter to apply when searching the directory.
	// +optional
	Filter string `json:"filter,omitempty"`

	// username attribute used for comparing user entries. This will be translated
	// and combined with the other filter as "(<attr>=<username>)".
	Username string `json:"username,omitempty"`

	// The following three fields are direct mappings of attributes on the user entry.

	// String representation of the user.
	IdAttr string `json:"idAttr,omitempty"`

	// Required. Attribute to map to Email
	EmailAttr string `json:"emailAttr,omitempty"`

	// Maps to display name of users. No default value.
	// +optional
	NameAttr string `json:"nameAttr,omitempty"`
}

// Group search queries for groups given a user entry.
type LDAPGroupSpec struct {
	// BaseDN to start the search from. It will translate to the query
	// "(&(objectClass=group)(member=<user uid>))".
	BaseDN string `json:"baseDn,omitempty"`

	// Optional filter to apply when searching the directory.
	Filter string `json:"filter,omitempty"`

	// Following two fields are used to match a user to a group. It adds an additional
	// requirement to the filter that an attribute in the group must match the user's
	// attribute value.
	UserAttr  string `json:"userAttr,omitempty"`
	GroupAttr string `json:"groupAttr,omitempty"`

	// Represents group name.
	// +optional
	NameAttr string `json:"nameAttr,omitempty"`
}

// LDAPConnectorSpec defines the desired state of LDAPConnector
type LDAPConnectorSpec struct {
	Name string `json:"name,omitempty"`

	Id string `json:"id,omitempty"`

	// Host and optional port of the LDAP server in the form "host:port".
	// If the port is not supplied, it will be guessed based on "insecureNoSSL",
	// and "startTLS" flags. 389 for insecure or StartTLS connections, 636
	// otherwise.
	Server string `json:"server,omitempty"`

	// The DN and password for an application service account. The connector uses
	// these credentials to search for users and groups. Not required if the LDAP
	// server provides access for anonymous auth.
	// Please note that if the bind password contains a `$`, it has to be saved in an
	// environment variable which should be given as the value to `bindPW`.
	// bindDN: uid=seviceaccount,cn=users,dc=example,dc=com
	// bindPW: password
	// +optional
	BindDN string `json:"bindDn,omitempty"`
	// +optional
	BindPW string `json:"bindPw,omitempty"`

	// +optional
	UsernamePrompt string `json:"usernamePrompt,omitempty"`

	// When connecting to the server, connect using the ldap:// protocol then issue
	// a StartTLS command. If unspecified, connections will use the ldaps:// protocol
	// +optional
	StartTLS bool `json:"startTLS,omitempty"`

	// Path to a trusted root certificate file. Default: use the host's root CA.
	// +optional
	RootCAData string `json:"rootCAData,omitempty"`

	// +optional
	User LDAPUserSpec `json:"user,omitempty"`

	// +optional
	Group LDAPGroupSpec `json:"group,omitempty"`
}

// LDAPConnectorStatus defines the observed state of LDAPConnector
type LDAPConnectorStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// LDAPConnector is the Schema for the ldapconnectors API
// +k8s:openapi-gen=true
type LDAPConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPConnectorSpec   `json:"spec,omitempty"`
	Status LDAPConnectorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// LDAPConnectorList contains a list of LDAPConnector
type LDAPConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LDAPConnector{}, &LDAPConnectorList{})
}
