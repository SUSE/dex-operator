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
	"testing"
)

func TestNewSharedPassword(t *testing.T) {
	password1 := NewSharedPassword("my-password1", "my-namespace")
	password1.Rand(10)
	t.Logf("Password generated 1: %s = %s", password1.Name, password1)
	if password1.Name != "my-namespace/my-password2" {
		t.Fatalf("Unexpected password name: %s", password2.Name)
	}

	password2 := NewSharedPassword("my-password2", "")
	password2.Rand(10)
	t.Logf("Password generated: %s = %s", password2.Name, password2)

	if password2.Name != "kube-system/my-password2" {
		t.Fatalf("Unexpected password name: %s", password2.Name)
	}
}
