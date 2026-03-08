// Copyright (c) 2021-2026, NVIDIA CORPORATION. All rights reserved.
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

func TestLoad(t *testing.T) {
	dir, err := ioutil.TempDir("", "stack_load_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	defPath := filepath.Join(dir, "stack.json")
	cfgPath := filepath.Join(dir, "config.json")
	defJSON := `{"name":"mystack","system":"host","type":"public","components":[]}`
	cfgJSON := `{"installDir":"/opt/stack","system":"host"}`
	if err := ioutil.WriteFile(defPath, []byte(defJSON), 0644); err != nil {
		t.Fatalf("write def file: %v", err)
	}
	if err := ioutil.WriteFile(cfgPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	c := Config{DefFilePath: defPath, ConfigFilePath: cfgPath, Loaded: false}
	if err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !c.Loaded {
		t.Error("Load: expected Loaded true")
	}
	if c.Data.StackDefinition == nil {
		t.Fatal("Load: StackDefinition is nil")
	}
	if c.Data.StackDefinition.Name != "mystack" || c.Data.StackDefinition.Type != "public" {
		t.Errorf("Load: StackDefinition = %+v", c.Data.StackDefinition)
	}
	if c.Data.StackConfig == nil {
		t.Fatal("Load: StackConfig is nil")
	}
	if c.Data.StackConfig.InstallDir != "/opt/stack" {
		t.Errorf("Load: StackConfig.InstallDir = %s", c.Data.StackConfig.InstallDir)
	}
}

func TestLoad_missingFile(t *testing.T) {
	c := Config{DefFilePath: "/nonexistent/def.json", ConfigFilePath: "/nonexistent/cfg.json"}
	err := c.Load()
	if err == nil {
		t.Fatal("Load: expected error for missing file")
	}
}

func TestUpdateRefs(t *testing.T) {
	c := Config{
		InstalledComponents: map[string]string{"foo": "/install/foo"},
		BuiltComponents:      map[string]string{"foo": "/build/foo"},
		SrcComponents:        map[string]string{"foo": "/src/foo"},
	}

	got, err := c.UpdateRefs("LIB=@ref:foo_install_dir@/lib")
	if err != nil {
		t.Fatalf("UpdateRefs: %v", err)
	}
	if want := "LIB=/install/foo/lib"; got != want {
		t.Errorf("UpdateRefs install_dir: got %q want %q", got, want)
	}

	got, err = c.UpdateRefs("BUILD=@ref:foo_build_dir@")
	if err != nil {
		t.Fatalf("UpdateRefs: %v", err)
	}
	if want := "BUILD=/build/foo"; got != want {
		t.Errorf("UpdateRefs build_dir: got %q want %q", got, want)
	}

	got, err = c.UpdateRefs("SRC=@ref:foo_src_dir@/x")
	if err != nil {
		t.Fatalf("UpdateRefs: %v", err)
	}
	if want := "SRC=/src/foo/x"; got != want {
		t.Errorf("UpdateRefs src_dir: got %q want %q", got, want)
	}
}

func TestUpdateRefs_noDelimiter(t *testing.T) {
	c := Config{InstalledComponents: map[string]string{"foo": "/install/foo"}}
	_, err := c.UpdateRefs("NO_REF_HERE")
	if err == nil {
		t.Fatal("UpdateRefs: expected error when token has no @ref:")
	}
}

func TestUpdateRefs_noUnderscore(t *testing.T) {
	c := Config{InstalledComponents: map[string]string{"foo": "/install/foo"}}
	_, err := c.UpdateRefs("X=@ref:invalidref@/y")
	if err == nil {
		t.Fatal("UpdateRefs: expected error when ref has no underscore")
	}
}
