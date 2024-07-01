// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package stack

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallStack(t *testing.T) {
	dummyCompName := "Comp1"
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create the temporary directory for testing: %s", err)
	}
	defer os.RemoveAll(testDir)

	components := []Component{
		{
			Name:                  dummyCompName,
			URL:                   "git@github.com:gvallee/c_hello_world.git",
			Branch:                "",
			BranchCheckoutPrelude: "",
			ConfigId:              dummyCompName,
			ConfigureDependency:   "",
			ConfigurePrelude:      "",
			ConfigureParams:       "",
			BuildEnv:              "",
		},
	}
	stackDef := StackDef{
		Name:       "test",
		System:     "host",
		Type:       "public",
		Components: components,
	}
	stackConfig := StackCfg{
		InstallDir: testDir,
		System:     "host",
	}
	stackData := Stack{
		Private:         false,
		BuildEnv:        nil,
		StackConfig:     &stackConfig,
		StackDefinition: &stackDef,
	}
	cfg := Config{
		DefFilePath:    "",
		ConfigFilePath: "",
		Loaded:         true,
		Data:           stackData,
	}

	err = cfg.InstallStack()
	if err != nil {
		t.Fatalf("stack installation failed: %s", err)
	}

	expectedBuildDir := filepath.Join(testDir, "test", "build", dummyCompName, "c_hello_world")
	expectedInstallDir := filepath.Join(testDir, "test", "install", dummyCompName)

	if cfg.BuiltComponents[dummyCompName] != expectedBuildDir {
		t.Fatalf("build directory is %s instead of %s", cfg.BuiltComponents[dummyCompName], expectedBuildDir)
	}

	if cfg.InstalledComponents[dummyCompName] != expectedInstallDir {
		t.Fatalf("install directory is %s instead of %s", cfg.InstalledComponents[dummyCompName], expectedInstallDir)
	}
}
