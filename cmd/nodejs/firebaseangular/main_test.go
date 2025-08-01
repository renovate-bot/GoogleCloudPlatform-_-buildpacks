// Copyright 2023 Google LLC
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

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	bmd "github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
)

const (
	mockLatestAngularAdapterVersion = "17.2.14"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		envs  []string
		want  int
	}{
		{
			name: "not a firebase apphosting app",
			files: map[string]string{
				"index.js":     "",
				"angular.json": "",
			},
			want: 100,
		},
		{
			name: "with angular config",
			files: map[string]string{
				"index.js":     "",
				"angular.json": "",
			},
			envs: []string{"X_GOOGLE_TARGET_PLATFORM=fah"},
			want: 0,
		},
		{
			name: "with angular config in app dir",
			files: map[string]string{
				"packages/foo/index.js":     "",
				"packages/foo/angular.json": "",
			},
			envs: []string{"GOOGLE_BUILDABLE=packages/foo", "X_GOOGLE_TARGET_PLATFORM=fah"},
			want: 0,
		},
		{
			name: "without angular config",
			files: map[string]string{
				"package.json": `{
				"dependencies": {
				},
				"devDependencies": {
				}
			}`, "package-lock.json": `{
				"packages": {
				}
			}`,
			},
			envs: []string{"X_GOOGLE_TARGET_PLATFORM=fah"},
			want: 100,
		},
		{
			name: "with angular builder dependency",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"build": "ng build"
					},
					"dependencies": {
						"@angular/core": "17.2.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/@angular/core": {
							"version": "17.2.0"
						},
						"node_modules/@angular-devkit/build-angular": {
							"version": "17.2.0"
						}
					}
				}`,
			},
			envs: []string{"X_GOOGLE_TARGET_PLATFORM=fah"},
			want: 0,
		},
		{
			name: "with apphosting:build script",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"apphosting:build": "adapter build"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/@angular/core": {
							"version": "17.2.0"
						},
						"node_modules/@angular-devkit/build-angular": {
							"version": "17.2.0"
						}
					}
				}`,
			},
			want: 100,
		},
		{
			name: "with apphosting.yaml buildcommand",
			files: map[string]string{
				"apphosting.yaml": `outputFiles:
  scripts:
    buildCommand: ng build`,
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name                string
		wantExitCode        int
		wantCommands        []string
		dontWantCommands    []string
		wantBuilderMetadata map[bmd.MetadataID]bmd.MetadataValue
		mocks               []*mockprocess.Mock
		files               map[string]string
	}{
		{
			name: "replace build script",
			files: map[string]string{
				"package.json": `{
				"scripts": {
					"build": "ng build"
				},
				"dependencies": {
					"@angular/core": "17.2.0"
				}
			}`,
				"package-lock.json": `{
				"packages": {
					"node_modules/@angular/core": {
						"version": "17.2.0"
					},
					"node_modules/@angular-devkit/build-angular": {
						"version": "17.2.0"
					}
				}
			}`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		},
		{
			name: "build script doesnt exist",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"@angular/core": "17.2.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/@angular/core": {
							"version": "17.2.0"
						},
					}
				}`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		},
		{
			name: "build script already set",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"build": "apphosting-adapter-angular-build"
					},
					"dependencies": {
						"@angular/core": "17.2.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/@angular/core": {
							"version": "17.2.0"
						},
					}
				}`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		},
		{
			name: "adapter already installed",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"build": "apphosting-adapter-angular-build"
					},
					"dependencies": {
						"@apphosting/adapter-angular": "17.2.5"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/@angular/core": {
							"version": "17.2.0"
						},
					}
				}`,
			},
			dontWantCommands: []string{
				"npm view @apphosting/adapter-angular version",
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		},
		{
			name: "error out if the builder version is below 17.2.0",
			files: map[string]string{
				"package.json": `{
				"dependencies": {
					"@angular/core": "17.2.0",
					"@angular-devkit/build-angular": "17.0.0"
				}
			}`,
				"package-lock.json": `{
				"packages": {
					"node_modules/@angular/core": {
						"version": "17.2.0"
					},
				}
			}`,
			},
			wantExitCode: 1,
		},
		{
			name: "should work if the version is exactly 17.2.0",
			files: map[string]string{
				"package.json": `{
				"dependencies": {
					"@angular/core": "17.2.0",
					"@angular-devkit/build-angular": "17.2.0"
				}
			}`,
				"package-lock.json": `{
				"packages": {
					"node_modules/@angular/core": {
						"version": "17.2.0"
					},
				}
			}`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
			wantExitCode: 0,
		},
		{
			name: "supports versions with constraints",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"build": "apphosting-adapter-angular-build"
					},
					"dependencies": {
						"@angular/core": "^17.0.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/@angular/core": {
							"version": "17.2.1"
						},
					}
				}`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		},
		{
			name: "read supported concrete version from package-lock.json",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"@angular/core": "15.0.0 - 17.2.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/@angular/core": {
							"version": "17.2.0"
						},
					}
				}`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		},
		{
			name: "read supported concrete version from pnpm-lock.yaml",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"@angular/core": "15.0.0 - 17.2.0"
					}
				}`,
				"pnpm-lock.yaml": `
dependencies:
  '@angular/core':
    version: 17.2.3(rxjs@7.8.1)(zone.js@0.14.4)
`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		},
		{
			name: "read supported concrete version from yaml.lock berry",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"@angular/core": "^17.1.0"
					}
				}`,
				"yarn.lock": `
	"@angular/core@npm:^17.1.0":
	version: 17.2.0`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		},
		{
			name: "read supported concrete version from yaml.lock classic",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"@angular/core": "^17.1.0"
					}
				}`,
				"yarn.lock": `
				@angular/core@^17.1.0:
	version: "17.2.0"
	`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		}, {
			name: "read supported concrete version from package.json with unsupported lock file format",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"@angular/core": "17.2.0"
					}
				}`,
				"pnpm-lock.yaml": `
unsupported:
  '@angular/core':
    version: 17.2.3(rxjs@7.8.1)(zone.js@0.14.4)
`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		}, {
			name: "read version range from package.json with unsupported lock file format",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"@angular/core": "17.2.0 - 18.0.0"
					}
				}`,
				"pnpm-lock.yaml": `
unsupported:
  '@angular/core':
    version: 17.2.3(rxjs@7.8.1)(zone.js@0.14.4)
`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
		}, {
			name: "set builder metadata correctly",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"build": "apphosting-adapter-angular-build"
					},
					"dependencies": {
						"@angular/core": "17.2.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/@angular/core": {
							"version": "17.2.0"
						}
					}
				}`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-angular@" + mockLatestAngularAdapterVersion,
			},
			wantBuilderMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    bmd.MetadataValue("angular"),
				bmd.FrameworkVersion: bmd.MetadataValue("17.2.0"),
				bmd.AdapterName:      bmd.MetadataValue("@apphosting/adapter-angular"),
				bmd.AdapterVersion:   bmd.MetadataValue(mockLatestAngularAdapterVersion),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithFiles(tc.files),
				bpt.WithExecMocks(tc.mocks...),
			}
			result, err := bpt.RunBuild(t, buildFn, opts...)
			if err != nil && tc.wantExitCode == 0 {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
			}

			if result.ExitCode != tc.wantExitCode {
				t.Errorf("build exit code mismatch, got: %d, want: %d", result.ExitCode, tc.wantExitCode)
			}

			for _, cmd := range tc.wantCommands {
				if !result.CommandExecuted(cmd) {
					t.Errorf("expected command %q to be executed, but it was not, build output: %s", cmd, result.Output)
				}
			}
			for _, cmd := range tc.dontWantCommands {
				if result.CommandExecuted(cmd) {
					t.Errorf("didn't expect command %q to be executed, but it was, build output: %s", cmd, result.Output)
				}
			}

			for id, m := range tc.wantBuilderMetadata {
				if m != result.MetadataAdded()[id] {
					t.Errorf("builder metadata %q mismatch, got: %s, want: %s", bmd.MetadataIDNames[id], result.MetadataAdded()[id], m)
				}
			}
		})
	}
}
