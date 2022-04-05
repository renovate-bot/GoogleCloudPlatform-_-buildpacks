// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: ".go files without go.mod, with target_platform",
			files: map[string]string{
				"main.go": "",
			},
			env:  []string{"X_GOOGLE_TARGET_PLATFORM=gae"},
			want: 0,
		},
		{
			name: ".go files without go.mod, without target_platform",
			files: map[string]string{
				"main.go": "",
			},
			want: 100,
		},
		{
			name: ".go files with go.mod",
			files: map[string]string{
				"go.mod":  "",
				"main.go": "",
			},
			env:  []string{"X_GOOGLE_TARGET_PLATFORM=gae"},
			want: 100,
		},
		{
			name:  "no files",
			files: map[string]string{},
			env:   []string{"X_GOOGLE_TARGET_PLATFORM=gae"},
			want:  100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}
