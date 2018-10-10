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
	"strings"
	"testing"
)

func TestNewSharedPassword(t *testing.T) {
	testEq := func(a, b []string) bool {
		// a simple (b ut not perfect) equality check
		return strings.Join(a, ",") == strings.Join(a, ",")
	}

	test1 := []string{"aaa", "bbb", "ccc"}
	test1Out := RemoveDuplicates(test1)
	if !testEq(test1, test1Out) {
		t.Logf("input: %+v", test1)
		t.Logf("-> output  : %+v", test1Out)
		t.Fatalf("unexpected output")
	}

	test2 := []string{"aaa", "bbb", "ccc", "bbb", "c"}
	test2Out := RemoveDuplicates(test2)
	expected2 := []string{"aaa", "bbb", "ccc", "c"}
	if !testEq(test1Out, expected2) {
		t.Logf("input: %+v", test2)
		t.Logf("-> output  : %+v", test2Out)
		t.Logf("-> expected: %+v", expected2)
		t.Fatalf("unexpected output")
	}

}
