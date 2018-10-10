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
	"testing"
)

func TestCreateDexConfigMap(t *testing.T) {

	// // The kubic-init configuration
	// cfg := config.KubicInitConfiguration{
	// }
	//
	// // create all the shared passwords we need
	// sharedPasswords, err := createSharedPasswords()
	// if err != nil {
	// 	t.Fatalf("Could not create a shared passwords for tests: %s", err)
	// }
	//
	// configMap, err := createConfigMap(&cfg, sharedPasswords)
	// if err != nil {
	// 	t.Fatalf("Could not generate ConfigMap for Dex: %s", err)
	// }
	//
	// configMapStr := string(configMap[:])
	// t.Logf("ConfigMap generated for Dex:\n%s\n", configMapStr)
	//
	// // TODO: perform more sophisticated checks...
}

func TestCreateDexDeployment(t *testing.T) {

	// // The kubic-init configuration
	// cfg := config.KubicInitConfiguration{}
	//
	// // create some fake certificate (just for getting the SHA256)
	// key, err := certutil.NewPrivateKey()
	// if err != nil {
	// 	t.Fatalf("Could not create a certificate for tests: %s", err)
	// }
	//
	// certCfg := certutil.Config{}
	// cert, err := certutil.NewSelfSignedCACert(certCfg, key)
	// if err != nil {
	// 	t.Fatalf("Could not create a certificate for tests: %s", err)
	// }
	//
	// // create all the shared passwords we need
	// sharedPasswords, err := createSharedPasswords()
	// if err != nil {
	// 	t.Fatalf("Could not create a shared passwords for tests: %s", err)
	// }
	//
	// // create the configmap
	// configMap, err := createConfigMap(&cfg, sharedPasswords)
	// if err != nil {
	// 	t.Fatalf("Could not generate configMap for Dex: %s", err)
	// }
	//
	// // and finally create the deployment
	// deployment, err := createDeployment(&cfg, configMap, cert)
	// if err != nil {
	// 	t.Fatalf("Could not generate Deployment  for Dex: %s", err)
	// }
	//
	// deploymentStr := string(deployment[:])
	// t.Logf("Deployment generated for Dex:\n%s\n", deploymentStr)
	//
	// // TODO: perform more sophisticated checks...
}
