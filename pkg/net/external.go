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

package net

import (
	"fmt"
	"net"

	kubicutil "github.com/kubic-project/dex-operator/pkg/util"
	utilnet "k8s.io/apimachinery/pkg/util/net"
)

const (
	defaultDNSDomain = "cluster.local"
)

// GetPublicAPIAddress gets a DNS name (or IP address)
// that can be used for reaching the API server
// TODO: fix this method: it will not work on containers
func GetPublicAPIAddress() (string, error) {
	localIP, err := utilnet.ChooseHostInterface()
	if err != nil {
		return "", err
	}
	return localIP.String(), nil
}

// GetInternalDNSName gets a FQDN DNS name in ther internal network for `name`
func GetServiceDNSName(obj kubicutil.ObjNamespacer) string {
	if len(obj.GetNamespace()) > 0 {
		return fmt.Sprintf("%s.%s.svc.%s", obj.GetName(), obj.GetNamespace(), defaultDNSDomain)
	}
	return fmt.Sprintf("%s.svc.%s", obj.GetName(), defaultDNSDomain)
}

// GetBindIP gets a valid IP address where we can bind
func GetBindIP() (net.IP, error) {
	defaultAddrStr := "0.0.0.0"

	defaultAddr := net.ParseIP(defaultAddrStr)
	bindIP, err := utilnet.ChooseBindAddress(defaultAddr)
	if err != nil {
		return nil, err
	}
	return bindIP, nil
}
