// Copyright (c) 2023-2026, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package module

import (
	"fmt"
	goerrs "github.com/gvallee/go_errs/pkg/goerrs"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultPermission  = 0775
	modulePrelude      = "#%Module1.0\n\n"
	conflictKeyword    = "conflict "
	requireKeyword     = "module load "
	setKeyword         = "set "
	setenvKeyword      = "setenv "
	prependPathKeyword = "prepend-path "
)

// Generate the file required to be able to use module for a specific software component.
// envVars specifies all the environment variables that needs to be set
// envLayout specifies the various environment variable to be preprended, the key is the target (e.g., PATH or LD_LIBRARY_PATH), the values path to a install directory
func Generate(path, copyright, customEnvVarPrefix, name string, requires []string, conflicts []string, vars map[string]string, envVars map[string]string, envLayout map[string][]string) error {
	modulefilePath := filepath.Join(path, name)

	content := modulePrelude

	content += copyright + "\n\n"

	for _, dep := range requires {
		content += requireKeyword + dep + "\n"
	}

	content += "\n"

	for _, conflict := range conflicts {
		content += conflictKeyword + conflict + "\n"
	}

	content += "\n"

	varKeys := make([]string, 0, len(vars))
	for varName := range vars {
		varKeys = append(varKeys, varName)
	}
	sort.Strings(varKeys)
	for _, varName := range varKeys {
		varValue := vars[varName]
		content += setKeyword + varName + " " + varValue + "\n"
	}

	content += "\n"

	envVarKeys := make([]string, 0, len(envVars))
	for varName := range envVars {
		envVarKeys = append(envVarKeys, varName)
	}
	sort.Strings(envVarKeys)
	for _, varName := range envVarKeys {
		varValue := envVars[varName]
		if customEnvVarPrefix == "" {
			content += setenvKeyword + varName + " " + varValue + "\n"
		} else {
			if strings.HasPrefix(varName, customEnvVarPrefix) {
				content += setenvKeyword + varName + " " + varValue + "\n"
			} else {
				content += setenvKeyword + customEnvVarPrefix + varName + " " + varValue + "\n"
			}
		}
	}

	content += "\n"

	envLayoutKeys := make([]string, 0, len(envLayout))
	for envvar := range envLayout {
		envLayoutKeys = append(envLayoutKeys, envvar)
	}
	sort.Strings(envLayoutKeys)
	for _, envvar := range envLayoutKeys {
		paths := envLayout[envvar]
		for _, layoutPath := range paths {
			content += prependPathKeyword + envvar + " " + layoutPath + "\n"
		}
	}

	err := ioutil.WriteFile(modulefilePath, []byte(content), defaultPermission)
	if err != nil {
		return goerrs.Wrap("Generate", "write_failed", fmt.Errorf("unable to write content of %s: %w", modulefilePath, err))
	}
	return nil
}
