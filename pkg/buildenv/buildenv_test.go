// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildenv

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gvallee/go_software_build/pkg/app"
	"github.com/gvallee/go_util/pkg/util"
)

func checkResultBuildEnv(testEnv Info, expectedEnv Info, t *testing.T) {
	t.Logf("checking for %s...", expectedEnv.SrcDir)
	if testEnv.SrcDir != expectedEnv.SrcDir {
		t.Fatalf("source dir has not been properly set; SrcDir is %s instead of %s", testEnv.SrcDir, expectedEnv.SrcDir)
	}
	t.Logf("checking for %s...", expectedEnv.SrcPath)
	if testEnv.SrcPath != expectedEnv.SrcPath {
		t.Fatalf("source path has not been properly set; SrcPath is %s instead of %s)", testEnv.SrcPath, expectedEnv.SrcPath)
	}

}

func TestDirURLGet(t *testing.T) {
	var testEnv Info
	var appInfo app.Info

	// The get operation is expected to result with a copy an entire directory into another directory
	dir1, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(dir1)
	t.Logf("Source directory is: %s", dir1)
	appInfo.Name = "testApp"
	appInfo.Source.URL = "file://" + dir1

	tempFile, err := ioutil.TempFile(dir1, "")
	if err != nil {
		t.Fatalf("unable to create temporary file: %s", err)
	}

	testEnv.BuildDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(testEnv.BuildDir)
	log.Printf("Build directory is: %s\n", testEnv.BuildDir)

	err = testEnv.Get(&appInfo)
	if err != nil {
		t.Fatalf("Get() failed: %s", err)
	}

	var expectedEnv Info
	expectedEnv.SrcDir = filepath.Join(testEnv.BuildDir, appInfo.Name, filepath.Base(dir1))
	expectedEnv.SrcPath = expectedEnv.SrcDir

	expectedFile := filepath.Join(expectedEnv.SrcDir, filepath.Base(tempFile.Name()))
	t.Logf("Checking if %s was correctly installed...", expectedFile)
	if !util.FileExists(expectedFile) {
		t.Fatalf("expected file %s does not exist", expectedFile)
	}

	checkResultBuildEnv(testEnv, expectedEnv, t)

	// We are supposed to be able to run twice in the row
	err = testEnv.Get(&appInfo)
	if err != nil {
		t.Fatalf("Get() failed: %s", err)
	}
}

func getHelloWorldSrc(t *testing.T) (*Info, *app.Info) {
	a := new(app.Info)
	a.Name = "helloworld"
	a.Source.URL = "https://github.com/gvallee/c_hello_world/archive/1.0.0.tar.gz"
	a.Version = "1.0.0"

	testEnv := new(Info)
	var err error
	// Create a temporary directory where files are downloaded
	testEnv.SrcDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}

	err = testEnv.Get(a)
	if err != nil {
		t.Fatalf("unable to download %s: %s", a.Source.URL, err)
	}

	return testEnv, a
}

func getHelloWorldGit(t *testing.T) (*Info, *app.Info) {
	a := new(app.Info)
	a.Name = "c_hello_world"
	a.Source.URL = "git@github.com:gvallee/c_hello_world.git"
	a.Source.Branch = "test"
	a.Version = "1.0.0"

	testEnv := new(Info)
	var err error
	testEnv.BuildDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}

	err = testEnv.Get(a)
	if err != nil {
		t.Fatalf("unable to download %s: %s", a.Source.URL, err)
	}

	return testEnv, a
}

func TestFileURLGet(t *testing.T) {
	testEnv, a := getHelloWorldSrc(t)
	defer os.RemoveAll(testEnv.SrcPath)

	var expectedEnv Info
	expectedEnv.SrcDir = filepath.Join(testEnv.BuildDir, a.Name)
	expectedEnv.SrcPath = filepath.Join(expectedEnv.SrcDir, filepath.Base(a.Source.URL))
	t.Logf("Checking if %s was correctly installed...", expectedEnv.SrcPath)
	if !util.FileExists(expectedEnv.SrcPath) {
		t.Fatalf("expected file %s does not exist", expectedEnv.SrcPath)
	}
}

func TestGitGet(t *testing.T) {
	testEnv, a := getHelloWorldGit(t)
	defer os.RemoveAll(testEnv.BuildDir)

	var expectedEnv Info
	expectedEnv.SrcPath = filepath.Join(testEnv.BuildDir, a.Name, a.Name)
	expectedEnv.SrcDir = filepath.Join(testEnv.BuildDir, a.Name, a.Name)
	t.Logf("Checking if %s was correctly installed...", expectedEnv.SrcPath)
	if !util.PathExists(expectedEnv.SrcPath) {
		t.Fatalf("expected file %s does not exist", expectedEnv.SrcPath)
	}

	t.Log("Checking that we got the correct branch...")
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("failed to find git: %s", err)
	}
	gitBranchCmd := exec.Command(gitBin, "branch")
	gitBranchCmd.Dir = expectedEnv.SrcDir
	var stderr, stdout bytes.Buffer
	gitBranchCmd.Stderr = &stderr
	gitBranchCmd.Stdout = &stdout
	err = gitBranchCmd.Run()
	if err != nil {
		t.Fatalf("command failed: %s - stdout: %s - stderr: %s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "* "+a.Source.Branch) {
		t.Fatalf("incorrect branch: %s", stdout.String())
	}

	checkResultBuildEnv(*testEnv, expectedEnv, t)
}

func TestMakeExtraArgs(t *testing.T) {
	testEnv, _ := getHelloWorldSrc(t)
	defer os.RemoveAll(testEnv.BuildDir)

	testEnv.MakeExtraArgs = append(testEnv.MakeExtraArgs, "CC=dummy")
	makefilePath := filepath.Join(testEnv.SrcDir, "Makefile")
	err := testEnv.RunMake(false, "", makefilePath, nil)
	if err == nil {
		t.Fatalf("test succeeded while expected to fail")
	}
}

func TestGetAppInstallDir(t *testing.T) {
	apps := []struct {
		name               string
		env                Info
		appInfo            app.Info
		expectedInstallDir string
	}{
		{
			name: "appNameSet",
			env: Info{
				InstallDir: "/one/dummy/path",
			},
			appInfo: app.Info{
				Name: "Application1",
				Source: app.SourceCode{
					URL: "https://something/toto.tar.gz",
				},
			},
			expectedInstallDir: "/one/dummy/path/Application1",
		},
		{
			name: "URL set only",
			env: Info{
				InstallDir: "/one/dummy/path",
			},
			appInfo: app.Info{
				Name: "",
				Source: app.SourceCode{
					URL: "https://something/application2.tar.bz2",
				},
			},
			expectedInstallDir: "/one/dummy/path/application2",
		},
		{
			name: "Git URL only",
			env: Info{
				InstallDir: "/one/dummy/path",
			},
			appInfo: app.Info{
				Name: "",
				Source: app.SourceCode{
					URL: "https://something/application3.git",
				},
			},
			expectedInstallDir: "/one/dummy/path/application3",
		},
	}

	for _, tt := range apps {
		result := tt.env.GetAppInstallDir(&tt.appInfo)
		if result != tt.expectedInstallDir {
			t.Fatalf("GetAppInstallDir is %s instead of %s", result, tt.expectedInstallDir)
		}
	}
}

func TestGetAppBuildDir(t *testing.T) {
	env := Info{BuildDir: "/tmp/build-root"}
	a := app.Info{Name: "myapp", Source: app.SourceCode{URL: "https://example.com/myapp.tar.gz"}}

	got := env.GetAppBuildDir(&a)
	want := filepath.Join(env.BuildDir, a.Name)
	if got != want {
		t.Fatalf("GetAppBuildDir = %s, want %s", got, want)
	}
}

func TestIsInstalled(t *testing.T) {
	installRoot, err := ioutil.TempDir("", "buildenv_install_")
	if err != nil {
		t.Fatalf("unable to create temporary install root: %s", err)
	}
	defer os.RemoveAll(installRoot)

	env := Info{InstallDir: installRoot}
	a := app.Info{Name: "installed-app"}

	if env.IsInstalled(&a) {
		t.Fatal("IsInstalled should be false before install dir exists")
	}

	if err := os.MkdirAll(filepath.Join(installRoot, a.Name), 0755); err != nil {
		t.Fatalf("unable to create app install dir: %s", err)
	}

	if !env.IsInstalled(&a) {
		t.Fatal("IsInstalled should be true once install dir exists")
	}
}

func TestGetEnvPathAndLDPath(t *testing.T) {
	env := Info{InstallDir: "/opt/test"}

	pathVal := env.GetEnvPath()
	if !strings.HasPrefix(pathVal, "/opt/test/bin:") {
		t.Fatalf("GetEnvPath() = %s, expected prefix /opt/test/bin:", pathVal)
	}

	ldPathVal := env.GetEnvLDPath()
	if !strings.HasPrefix(ldPathVal, "/opt/test/lib:") {
		t.Fatalf("GetEnvLDPath() = %s, expected prefix /opt/test/lib:", ldPathVal)
	}
}

func TestLookPath(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "buildenv_lookpath_")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath := filepath.Join(tmpDir, "mybin")
	if err := ioutil.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("unable to create fake binary: %s", err)
	}

	env := Info{Env: []string{"PATH=" + tmpDir + ":/usr/bin"}}
	resolved := env.lookPath("mybin")
	if resolved != binaryPath {
		t.Fatalf("lookPath returned %s instead of %s", resolved, binaryPath)
	}

	notFound := env.lookPath("missing-bin")
	if notFound != "missing-bin" {
		t.Fatalf("lookPath should return original name for missing binaries, got %s", notFound)
	}
}

func TestInstall_noInstallCommand(t *testing.T) {
	env := Info{}
	a := app.Info{Name: "noop", InstallCmd: ""}

	if err := env.Install(&a); err != nil {
		t.Fatalf("Install() should succeed when InstallCmd is empty: %s", err)
	}
}

func TestInstall_withCommand(t *testing.T) {
	srcDir, err := ioutil.TempDir("", "buildenv_install_cmd_src_")
	if err != nil {
		t.Fatalf("unable to create source directory: %s", err)
	}
	defer os.RemoveAll(srcDir)

	targetFile := filepath.Join(srcDir, "installed.txt")
	env := Info{SrcDir: srcDir}
	a := app.Info{Name: "cmdinstall", InstallCmd: "touch " + targetFile}

	if err := env.Install(&a); err != nil {
		t.Fatalf("Install() failed: %s", err)
	}
	if !util.FileExists(targetFile) {
		t.Fatalf("expected file %s does not exist", targetFile)
	}
}

func TestInitCreatesDirectories(t *testing.T) {
	baseDir, err := ioutil.TempDir("", "buildenv_init_")
	if err != nil {
		t.Fatalf("unable to create base directory: %s", err)
	}
	defer os.RemoveAll(baseDir)

	env := Info{
		ScratchDir: filepath.Join(baseDir, "scratch"),
		BuildDir:   filepath.Join(baseDir, "build"),
		InstallDir: filepath.Join(baseDir, "install"),
	}

	if err := env.Init(); err != nil {
		t.Fatalf("Init() failed: %s", err)
	}

	if !util.PathExists(env.ScratchDir) {
		t.Fatalf("ScratchDir was not created: %s", env.ScratchDir)
	}
	if !util.PathExists(env.BuildDir) {
		t.Fatalf("BuildDir was not created: %s", env.BuildDir)
	}
	if !util.PathExists(env.InstallDir) {
		t.Fatalf("InstallDir was not created: %s", env.InstallDir)
	}
}

func TestGetPathFromFileURL(t *testing.T) {
	path, err := getPathFromFileURL("file:///tmp/archive.tar.gz")
	if err != nil {
		t.Fatalf("getPathFromFileURL() failed: %s", err)
	}
	if path != "/tmp/archive.tar.gz" {
		t.Fatalf("unexpected path: %s", path)
	}

	if _, err := getPathFromFileURL("https://example.com/archive.tar.gz"); err == nil {
		t.Fatal("expected error for non-file URL")
	}

	if _, err := getPathFromFileURL("file://"); err == nil {
		t.Fatal("expected error for empty file URL path")
	}
}

func TestCopyTarball(t *testing.T) {
	buildDir, err := ioutil.TempDir("", "buildenv_copy_build_")
	if err != nil {
		t.Fatalf("unable to create build dir: %s", err)
	}
	defer os.RemoveAll(buildDir)

	srcDir, err := ioutil.TempDir("", "buildenv_copy_src_")
	if err != nil {
		t.Fatalf("unable to create src dir: %s", err)
	}
	defer os.RemoveAll(srcDir)

	srcTarball := filepath.Join(srcDir, "dummy.tar.gz")
	if err := ioutil.WriteFile(srcTarball, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("unable to create source tarball: %s", err)
	}

	env := Info{BuildDir: buildDir}
	a := app.Info{Name: "dummy", Source: app.SourceCode{URL: "file://" + srcTarball}}

	if err := env.copyTarball(&a); err != nil {
		t.Fatalf("copyTarball() failed: %s", err)
	}

	expectedPath := filepath.Join(buildDir, a.Name, filepath.Base(srcTarball))
	if env.SrcPath != expectedPath {
		t.Fatalf("SrcPath is %s instead of %s", env.SrcPath, expectedPath)
	}
	if env.SrcDir != filepath.Join(buildDir, a.Name) {
		t.Fatalf("SrcDir is %s instead of %s", env.SrcDir, filepath.Join(buildDir, a.Name))
	}
	if !util.FileExists(expectedPath) {
		t.Fatalf("expected copied tarball %s does not exist", expectedPath)
	}
}
