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
	"bytes"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"text/template"
)

// ParseTemplate processes a text/template, doing some replacements
func ParseTemplate(templateStr string, replacements interface{}) (string, error) {

	indent := func(spaces int, v string) string {
		pad := strings.Repeat(" ", spaces)
		return strings.Replace(v, "\n", "\n"+pad, -1)
	}

	replace := func(old, new, src string) string {
		return strings.Replace(src, old, new, -1)
	}

	base64encode := func(v string) string {
		return base64.StdEncoding.EncodeToString([]byte(v))
	}

	base64decode := func(v string) string {
		data, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err.Error()
		}
		return string(data)
	}

	safePath := func(v string) string {
		replacer := strings.NewReplacer(" ", `\ `, ":", `\:`)
		return replacer.Replace(v)
	}

	// some custom functions
	funcMap := template.FuncMap{
		"indent":       indent,
		"replace":      replace,
		"base64encode": base64encode,
		"base64decode": base64decode,
		"url64encode": func(v string) string {
			return URL64encode(v)
		},
		"url64decode": func(v string) string {
			return URL64decode(v)
		},
		"safeYAMLId": func(v string) string {
			return SafeId(v)
		},
		"safePath": safePath,
		"basename": func(v string) string {
			return path.Base(v)
		},
		"dirname": func(v string) string {
			return path.Dir(v)
		},
	}

	var buf bytes.Buffer
	tmpl, err := template.New("template").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("error when parsing template: %v", err)
	}

	err = tmpl.Execute(&buf, replacements)
	if err != nil {
		return "", fmt.Errorf("error when executing template: %v", err)
	}

	contents := buf.String()
	if err != nil {
		return "", fmt.Errorf("error when parsing AutoYaST template: %v", err)
	}

	return contents, nil
}
