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

// Package php contains PHP buildpack library code.
package php

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

const (
	// composerJSON is the name of the Composer package descriptor file.
	composerJSON = "composer.json"
	// composerLock is the name of the Composer lock file.
	composerLock = "composer.lock"
	// Vendor is the name of the Composer vendor directory.
	Vendor = "vendor"

	phpVersionKey     = "php_version"
	dependencyHashKey = "dependency_hash"

	composerVersionKey = "php"

	// PHPIni is the content of the php.ini config file
	PHPIni = `
; Copyright 2022 Google Inc.
;
; Licensed under the Apache License, Version 2.0 (the "License");
; you may not use this file except in compliance with the License.
; You may obtain a copy of the License at
;
;     http://www.apache.org/licenses/LICENSE-2.0
;
; Unless required by applicable law or agreed to in writing, software
; distributed under the License is distributed on an "AS IS" BASIS,
; WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
; See the License for the specific language governing permissions and
; limitations under the License.

expose_php = Off
memory_limit = -1
max_execution_time = 0

;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
; Error handling and logging, based on php.ini-production. ;
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;

error_reporting = E_ALL & ~E_DEPRECATED & ~E_STRICT
display_errors = Off
display_startup_errors = Off
log_errors = On
log_errors_max_len = 0
ignore_repeated_errors = Off
ignore_repeated_source = Off
html_errors = Off
zend.assertions = -1
;; Enable maximum file sizes up to Front-End limits.
upload_max_filesize = 32M
post_max_size = 32M
`
)

type composerScriptsJSON struct {
	GCPBuild string `json:"gcp-build"`
}

// ComposerJSON represents the contents of a composer.json file.
type ComposerJSON struct {
	Require map[string]string   `json:"require"`
	Scripts composerScriptsJSON `json:"scripts"`
}

// SupportsAppEngineApis is a function that returns true if App Engine API access is enabled
func SupportsAppEngineApis(ctx *gcp.Context) (bool, error) {
	if os.Getenv(env.Runtime) == "php55" {
		return true, nil
	}

	return appengine.ApisEnabled(ctx)
}

// ReadComposerJSON returns the deserialized composer.json from the given dir. Empty dir uses the current working directory.
func ReadComposerJSON(dir string) (*ComposerJSON, error) {
	f := filepath.Join(dir, composerJSON)
	rawcjs, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, gcp.InternalErrorf("reading %s: %v", composerJSON, err)
	}

	var cjs ComposerJSON
	if err := json.Unmarshal(rawcjs, &cjs); err != nil {
		return nil, gcp.UserErrorf("unmarshalling %s: %v", composerJSON, err)
	}
	return &cjs, nil
}

// version returns the installed version of PHP.
func version(ctx *gcp.Context) string {
	result := ctx.Exec([]string{"php", "-r", "echo PHP_VERSION;"})
	return result.Stdout
}

// checkCache checks whether cached dependencies exist and match.
func checkCache(ctx *gcp.Context, l *libcnb.Layer, opts ...cache.Option) (bool, error) {
	currentPHPVersion := version(ctx)
	opts = append(opts, cache.WithStrings(currentPHPVersion))
	currentDependencyHash, err := cache.Hash(ctx, opts...)
	if err != nil {
		return false, fmt.Errorf("computing dependency hash: %v", err)
	}

	// Perform install, skipping if the dependency hash matches existing metadata.
	metaDependencyHash := ctx.GetMetadata(l, dependencyHashKey)
	ctx.Debugf("Current dependency hash: %q", currentDependencyHash)
	ctx.Debugf("  Cache dependency hash: %q", metaDependencyHash)
	if currentDependencyHash == metaDependencyHash {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, nil
	}

	if metaDependencyHash == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}
	ctx.Logf("Installing application dependencies.")

	// Update the layer metadata.
	ctx.SetMetadata(l, dependencyHashKey, currentDependencyHash)
	ctx.SetMetadata(l, phpVersionKey, currentPHPVersion)

	return false, nil
}

// composerInstall runs `composer install` with the given flags.
func composerInstall(ctx *gcp.Context, flags []string) {
	cmd := append([]string{"composer", "install"}, flags...)
	ctx.Exec(cmd, gcp.WithUserAttribution)
}

// ComposerInstall runs `composer install`, using the cache iff a lock file is present.
// It creates a layer, so it returns the layer so that the caller may further modify it
// if they desire.
func ComposerInstall(ctx *gcp.Context, cacheTag string) (*libcnb.Layer, error) {
	// We don't install dev dependencies (i.e. we pass --no-dev to composer) because doing so has caused
	// problems for customers in the past. For more information see these links:
	//   https://github.com/GoogleCloudPlatform/php-docs-samples/issues/736
	//   https://github.com/GoogleCloudPlatform/runtimes-common/pull/763
	//   https://github.com/GoogleCloudPlatform/runtimes-common/commit/6c4970f609d80f9436ac58ae272cfcc6bcd57143
	flags := []string{"--no-dev", "--no-progress", "--no-interaction", "--optimize-autoloader"}

	if err := ctx.RemoveAll(Vendor); err != nil {
		return nil, err
	}
	l, err := ctx.Layer("composer", gcp.CacheLayer)
	if err != nil {
		return nil, fmt.Errorf("creating layer: %w", err)
	}
	layerVendor := filepath.Join(l.Path, Vendor)

	composerLockExists, err := ctx.FileExists(composerLock)
	if err != nil {
		return nil, err
	}
	// If there's no composer.lock then don't attempt to cache. We'd have to cache using composer.json,
	// which could result in outdated dependencies if the version constraints in composer.json resolve
	// to newer versions in the future.
	if !composerLockExists {
		ctx.Logf("*** Improve build performance by generating and committing %s.", composerLock)
		composerInstall(ctx, flags)
		return l, nil
	}

	cached, err := checkCache(ctx, l, cache.WithFiles(composerJSON, composerLock))
	if err != nil {
		return l, fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(cacheTag)

		// PHP expects the vendor/ directory to be in the application directory.
		ctx.Exec([]string{"cp", "--archive", layerVendor, Vendor}, gcp.WithUserTimingAttribution)
	} else {
		ctx.CacheMiss(cacheTag)
		// Clear layer so we don't end up with outdated dependencies (e.g. something was removed from composer.json).
		if err := ctx.ClearLayer(l); err != nil {
			return nil, fmt.Errorf("clearing layer %q: %w", l.Name, err)
		}
		composerInstall(ctx, flags)

		// Ensure vendor exists even if no dependencies were installed.
		if err := ctx.MkdirAll(Vendor, 0755); err != nil {
			return nil, err
		}
		ctx.Exec([]string{"cp", "--archive", Vendor, layerVendor}, gcp.WithUserTimingAttribution)
	}

	return l, nil
}

// ComposerRequire runs `composer require` with the given packages. It expects packages to
// be specified as `composer require` would expect them on the command line, for example
// "myorg/mypackage:^0.7". It does no caching.
func ComposerRequire(ctx *gcp.Context, packages []string) {
	cmd := append([]string{"composer", "require", "--no-progress", "--no-interaction"}, packages...)
	ctx.Exec(cmd, gcp.WithUserAttribution)
}

// ExtractVersion extracts the php version from the environment, composer.json.
func ExtractVersion(ctx *gcp.Context) (string, error) {
	// get the runtime version from env.RuntimeVersion
	if v := os.Getenv(env.RuntimeVersion); v != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, v)
		return v, nil
	}

	// get the runtime version from the composer.json file
	composerFilePath := filepath.Join(ctx.ApplicationRoot(), composerJSON)
	composerFileExists, err := ctx.FileExists(composerFilePath)
	if err != nil {
		return "", err
	}
	if composerFileExists {
		v, err := composerFileVersion(ctx)
		if err != nil {
			return "", err
		}
		if v != "" {
			ctx.Logf("Using php version from %s %s: %s", composerJSON, composerVersionKey, v)
			return v, nil
		}
	}

	return "", nil
}

// composerFileVersion extracts the version number from composer.json. returns an error in
// case the version cannot be read.
func composerFileVersion(ctx *gcp.Context) (string, error) {
	cjs, err := ReadComposerJSON(ctx.ApplicationRoot())
	if err != nil {
		return "", err
	}

	// check if composer json has specified php version.
	v, ok := cjs.Require[composerVersionKey]
	if !ok {
		ctx.Logf("composer.json exists but does not specify a php version")
		return "", nil
	}

	return v, nil
}
