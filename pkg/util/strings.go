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
	"encoding/base64"
	"strings"
)

// SafeId returns a safe ID (for example, for using in YAML)
// ie, "Something:6000/ddd" becomes "something-6000-ddd"
func SafeId(s string) string {
	replacer := strings.NewReplacer(" ", "-", ":", "-", "/", "-", ".", "-")
	return replacer.Replace(strings.ToLower(s))
}

func URL64encode(v string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(v))
}

func URL64decode(v string) string {
	data, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func RemoveDuplicates(in []string) []string {
	processed := map[string]struct{}{}

	res := []string{}
	for _, s := range in {
		if _, found := processed[s]; !found {
			processed[s] = struct{}{}
			res = append(res, s)
		}
	}

	return res
}
