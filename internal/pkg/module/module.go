// Copyright (c) 2023, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package module

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

const (
	defaultPermission  = 0766
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
func Generate(path string, copyright string, name string, requires []string, conflicts []string, vars map[string]string, envVars map[string]string, envLayout map[string][]string) error {
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

	for varName, varValue := range vars {
		content += setKeyword + varName + " " + varValue + "\n"
	}

	content += "\n"

	for varName, varValue := range envVars {
		content += setenvKeyword + varName + " " + varValue + "\n"
	}

	content += "\n"

	for envvar, paths := range envLayout {
		for _, path := range paths {
			content += prependPathKeyword + envvar + " " + path + "\n"
		}
	}

	err := ioutil.WriteFile(modulefilePath, []byte(content), defaultPermission)
	if err != nil {
		return fmt.Errorf("unable to write content of %s: %w", modulefilePath, err)
	}
	return nil
}
