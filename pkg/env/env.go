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

// Package env specifies environment variables used to configure buildpack behavior.
package env

import (
	"fmt"
	"os"
	"strconv"
)

const (
	// Runtime is an env var used constrain autodetection in runtime buildpacks or to set runtime name in App Engine buildpacks.
	// Runtime must be respected by each runtime buildpack.
	// Example: `nodejs` will cause the nodejs/runtime buildpack to opt-in.
	Runtime = "GOOGLE_RUNTIME"

	// RuntimeVersion is an env var used to specify which runtime version to install.
	// RuntimeVersion must be respected by each runtime buildpack.
	// Example: `13.7.0` for Node.js, `1.14.1` for Go.
	RuntimeVersion = "GOOGLE_RUNTIME_VERSION"

	// DebugMode enables more verbose logging. The value is unused; only the presense of the env var is required to enable.
	DebugMode = "GOOGLE_DEBUG"

	// DevMode is an env var used to enable development mode in buildpacks.
	// DevMode should be respected by all buildpacks that are not product-specific.
	// Example: `true`, `True`, `1` will enable development mode.
	DevMode = "GOOGLE_DEVMODE"

	// Entrypoint is an env var used to override the default entrypoint.
	// Entrypoint should be respected by at least one buildpack in builders that are not product-specific.
	// Example: `gunicorn -p :8080 main:app` for Python.
	Entrypoint = "GOOGLE_ENTRYPOINT"

	// KeepSource is an env var used to keep the source in the final image.
	// Buildpacks for compiled languages (e.g., Go, Java, .Net) will typically strip out the source.
	KeepSource = "GOOGLE_KEEP_SOURCE"

	// Buildable is an env var used to specify the buildable unit to build.
	// Buildable should be respected by buildpacks that build source.
	// Example: `./maindir` for Go will build the package rooted at maindir.
	Buildable = "GOOGLE_BUILDABLE"

	// GAEMain is an env var used to specify path or fully qualified package name of the main package in App Engine buildpacks.
	// Behavior: In Go, the value is cleaned up and passed on to subsequent buildpacks as GOOGLE_BUILDABLE.
	GAEMain = "GAE_YAML_MAIN"

	// FunctionTarget is an env var used to specify function name.
	// FunctionTarget must be respected by all functions-framework buildpacks.
	// Example: `helloWorld` or any exported function name.
	FunctionTarget = "FUNCTION_TARGET"

	// FunctionSource is an env var used to specify function source location.
	// FunctionSource must be respected by all functions-framework buildpacks.
	// Example: `./path/to/source` will build the function at the specfied path.
	FunctionSource = "FUNCTION_SOURCE"

	// FunctionSignatureType is an env var used to specify function signature type.
	// FunctionSignatureType must be respected by all functions-framework buildpacks.
	// Example: `http` for HTTP-triggered functions or `event` for event-triggered functions.
	FunctionSignatureType = "FUNCTION_SIGNATURE_TYPE"
)

// IsDebugMode returns true if the buildpack debug mode is enabled.
func IsDebugMode() (bool, error) {
	val, found := os.LookupEnv(DebugMode)
	if !found {
		return false, nil
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("parsing %s: %v", DebugMode, err)
	}
	return parsed, nil
}
