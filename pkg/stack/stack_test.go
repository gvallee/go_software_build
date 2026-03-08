// Copyright (c) 2021-2026, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package stack

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestUpdateRefs_unknownComponent(t *testing.T) {
	c := Config{InstalledComponents: map[string]string{"foo": "/install/foo"}}
	_, err := c.UpdateRefs("X=@ref:bar_install_dir@/y")
	if err == nil {
		t.Fatal("UpdateRefs: expected error for unknown component")
	}
}

func TestUpdateRefs_unsupportedRefType(t *testing.T) {
	c := Config{InstalledComponents: map[string]string{"foo": "/install/foo"}}
	_, err := c.UpdateRefs("X=@ref:foo_unknown_ref@/y")
	if err == nil {
		t.Fatal("UpdateRefs: expected error for unsupported ref type")
	}
}

func TestUpdateRefs_missingBuildDir(t *testing.T) {
	c := Config{
		InstalledComponents: map[string]string{"foo": "/install/foo"},
		BuiltComponents:      map[string]string{},
	}
	_, err := c.UpdateRefs("X=@ref:foo_build_dir@/y")
	if err == nil {
		t.Fatal("UpdateRefs: expected error for missing build dir")
	}
}

func TestUpdateRefs_missingSrcDir(t *testing.T) {
	c := Config{
		InstalledComponents: map[string]string{"foo": "/install/foo"},
		SrcComponents:        map[string]string{},
	}
	_, err := c.UpdateRefs("X=@ref:foo_src_dir@/y")
	if err == nil {
		t.Fatal("UpdateRefs: expected error for missing src dir")
	}
}

func TestGetCompBuildDir(t *testing.T) {
	baseDir, err := ioutil.TempDir("", "stack_builddir_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	compName := "foo"
	buildRoot := filepath.Join(baseDir, "build", compName)
	if err := os.MkdirAll(filepath.Join(buildRoot, "foo-build"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := GetCompBuildDir(baseDir, compName)
	if err != nil {
		t.Fatalf("GetCompBuildDir: %v", err)
	}
	want := filepath.Join(buildRoot, "foo-build")
	if got != want {
		t.Fatalf("GetCompBuildDir = %s, want %s", got, want)
	}
}

func TestGetCompBuildDir_noSubdir(t *testing.T) {
	baseDir, err := ioutil.TempDir("", "stack_builddir_empty_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	compName := "foo"
	buildRoot := filepath.Join(baseDir, "build", compName)
	if err := os.MkdirAll(buildRoot, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := GetCompBuildDir(baseDir, compName); err == nil {
		t.Fatal("GetCompBuildDir: expected error when no build subdirectory exists")
	}
}

func TestGetCompSrcDir(t *testing.T) {
	baseDir, err := ioutil.TempDir("", "stack_srcdir_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	compName := "foo"
	srcRoot := filepath.Join(baseDir, "src")
	srcDir := filepath.Join(srcRoot, "foo-1.0.0")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := GetCompSrcDir(baseDir, compName)
	if err != nil {
		t.Fatalf("GetCompSrcDir: %v", err)
	}
	if got != srcDir {
		t.Fatalf("GetCompSrcDir = %s, want %s", got, srcDir)
	}
}

func TestGetCompSrcDir_noMatch(t *testing.T) {
	baseDir, err := ioutil.TempDir("", "stack_srcdir_nomatch_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	if err := os.MkdirAll(filepath.Join(baseDir, "src", "bar-1.0.0"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := GetCompSrcDir(baseDir, "foo"); err == nil {
		t.Fatal("GetCompSrcDir: expected error when no matching source directory exists")
	}
}

func TestGenerateModules(t *testing.T) {
	baseDir, err := ioutil.TempDir("", "stack_modules_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	stackName := "mystack"
	stackBase := filepath.Join(baseDir, stackName)
	compName := "comp1"
	compInstall := filepath.Join(stackBase, "install", compName)
	if err := os.MkdirAll(filepath.Join(compInstall, "bin"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(compInstall, "lib"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(compInstall, "include"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(compInstall, "man"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(compInstall, "lib", "pkgconfig"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(stackBase, "build", compName, "builddir"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	defPath := filepath.Join(baseDir, "stack.json")
	cfgPath := filepath.Join(baseDir, "config.json")
	defJSON := `{"name":"` + stackName + `","system":"host","type":"public","components":[{"name":"` + compName + `","configure_dependency":"dep1,dep2"}]}`
	cfgJSON := `{"installDir":"` + baseDir + `","system":"host"}`
	if err := ioutil.WriteFile(defPath, []byte(defJSON), 0644); err != nil {
		t.Fatalf("write def file: %v", err)
	}
	if err := ioutil.WriteFile(cfgPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	c := Config{DefFilePath: defPath, ConfigFilePath: cfgPath}

	if err := c.GenerateModules("copyright", "MYP_"); err != nil {
		t.Fatalf("GenerateModules failed: %v", err)
	}

	modulefile := filepath.Join(stackBase, "modulefiles", compName)
	if !fileExists(modulefile) {
		t.Fatalf("expected modulefile %s to exist", modulefile)
	}
	content, err := ioutil.ReadFile(modulefile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "module load dep1\n") || !strings.Contains(contentStr, "module load dep2\n") {
		t.Fatal("expected dependency module load entries")
	}
	if !strings.Contains(contentStr, "setenv MYP_"+strings.ToUpper(compName)+"_DIR ") {
		t.Fatal("expected prefixed component DIR environment variable")
	}
	if !strings.Contains(contentStr, "prepend-path PATH ") {
		t.Fatal("expected PATH prepend entry")
	}
}

func TestExportAndImport(t *testing.T) {
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar not available")
	}

	baseDir, err := ioutil.TempDir("", "stack_export_import_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	stackName := "expstack"
	stackBase := filepath.Join(baseDir, stackName)
	installDir := filepath.Join(stackBase, "install")
	if err := os.MkdirAll(filepath.Join(installDir, "comp1", "bin"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	marker := filepath.Join(installDir, "comp1", "bin", "tool")
	if err := ioutil.WriteFile(marker, []byte("dummy"), 0644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	defPath := filepath.Join(baseDir, "stack.json")
	cfgPath := filepath.Join(baseDir, "config.json")
	defJSON := `{"name":"` + stackName + `","system":"host","type":"public","components":[]}`
	cfgJSON := `{"installDir":"` + baseDir + `","system":"host"}`
	if err := ioutil.WriteFile(defPath, []byte(defJSON), 0644); err != nil {
		t.Fatalf("write def file: %v", err)
	}
	if err := ioutil.WriteFile(cfgPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	c := Config{DefFilePath: defPath, ConfigFilePath: cfgPath}

	if err := c.Export(); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	tarball := filepath.Join(stackBase, stackName+".tar.bz2")
	if !fileExists(tarball) {
		t.Fatalf("expected tarball %s to exist", tarball)
	}

	if err := os.RemoveAll(installDir); err != nil {
		t.Fatalf("remove install dir: %v", err)
	}

	if err := c.Import(tarball); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if !fileExists(marker) {
		t.Fatalf("expected marker %s to be restored after import", marker)
	}

}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
