// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package builder

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gvallee/go_exec/pkg/advexec"
	"github.com/gvallee/go_util/pkg/util"
)

func setBuilder(t *testing.T) (*Builder, func()) {
	b := new(Builder)

	var err error
	b.Env.ScratchDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create scratch directory: %s", err)
	}
	t.Logf("Scratch directory: %s", b.Env.ScratchDir)

	b.Env.InstallDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create install directory: %s", err)
	}
	t.Logf("Install directory: %s", b.Env.InstallDir)

	b.Env.BuildDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create build directory: %s", err)
	}
	t.Logf("Build directory: %s", b.Env.BuildDir)

	// Create a directory where software is downloaded
	b.Env.SrcDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create src directory: %s", err)
	}
	t.Logf("Src directory: %s", b.Env.SrcDir)

	t.Cleanup(func() {
		os.RemoveAll(b.Env.ScratchDir)
		os.RemoveAll(b.Env.InstallDir)
		os.RemoveAll(b.Env.BuildDir)
		os.RemoveAll(b.Env.SrcDir)
	})

	return b, func() {}
}

// requireBinaryExists fails the test if the expected binary path does not exist.
func requireBinaryExists(t *testing.T, installDir, appName, binName string) {
	t.Helper()
	p := filepath.Join(installDir, appName, "bin", binName)
	if !util.FileExists(p) {
		t.Fatalf("expected binary %s does not exist", p)
	}
}

func TestInstallFromAutotoolsRelease(t *testing.T) {
	b, _ := setBuilder(t)
	t.Logf("Build directory: %s", b.Env.BuildDir)
	t.Logf("Source directory: %s", b.Env.SrcDir)
	b.App.Name = "c_hello_world"
	b.App.Source.URL = "https://github.com/gvallee/c_hello_world/releases/download/v1.0.1/c_hello_world-1.0.1.tar.gz"
	b.App.Version = "1.0.1"

	if err := b.Load(false); err != nil {
		t.Fatalf("unable to load the builder: %s", err)
	}
	if b.Configure == nil {
		t.Fatal("builder configure is undefined")
	}

	res := b.Install()
	if res.Err != nil {
		t.Fatalf("unable to install the software package: %s", res.Err)
	}
	requireBinaryExists(t, b.Env.InstallDir, b.App.Name, "helloworld")
}

func TestBuilderEnv(t *testing.T) {
	b, _ := setBuilder(t)
	t.Logf("Build directory: %s", b.Env.BuildDir)
	t.Logf("Source directory: %s", b.Env.SrcDir)
	b.App.Name = "helloworld"
	b.App.Source.URL = "https://github.com/gvallee/c_hello_world/archive/1.0.0.tar.gz"
	b.App.Version = "1.0.0"
	b.Env.Env = append(b.Env.Env, "CC=/dummy/toto")

	if err := b.Load(false); err != nil {
		t.Fatalf("unable to load the builder: %s", err)
	}
	if b.Configure == nil {
		t.Fatal("builder configure is undefined")
	}

	res := b.Install()
	if res.Err == nil {
		t.Fatal("install succeeded even when specifying an invalid C compiler through the environment")
	}
	if res.Stderr != "" {
		t.Logf("Install failed as expected; stderr: %s", res.Stderr)
	}
}

func TestInstallFromSource(t *testing.T) {
	b, _ := setBuilder(t)
	t.Logf("Build directory: %s", b.Env.BuildDir)
	t.Logf("Source directory: %s", b.Env.SrcDir)
	b.App.Name = "helloworld"
	b.App.Source.URL = "https://github.com/gvallee/c_hello_world/archive/1.0.0.tar.gz"
	b.App.Version = "1.0.0"

	if err := b.Load(false); err != nil {
		t.Fatalf("unable to load the builder: %s", err)
	}
	if b.Configure == nil {
		t.Fatal("builder configure is undefined")
	}

	res := b.Install()
	if res.Err != nil {
		t.Fatalf("unable to install the software package: %s", res.Err)
	}
	requireBinaryExists(t, b.Env.InstallDir, b.App.Name, "helloworld")
}

func TestPersistentBuildFromLocalTarball(t *testing.T) {
	url := "https://github.com/gvallee/c_hello_world/archive/1.0.0.tar.gz"
	tarballFilename := "1.0.0.tar.gz"
	wgetBin, err := exec.LookPath("wget")
	if err != nil {
		t.Skip("wget not available skipping test")
	}

	downloadDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(downloadDir)

	downloadCmd := new(advexec.Advcmd)
	downloadCmd.BinPath = wgetBin
	downloadCmd.CmdArgs = []string{url}
	downloadCmd.ExecDir = downloadDir
	res := downloadCmd.Run()
	if res.Err != nil {
		t.Fatalf("unable to download tarball")
	}

	downloadedTarball := filepath.Join(downloadDir, tarballFilename)
	if !util.FileExists(downloadedTarball) {
		t.Fatalf("unable to find tarball (expected: %s)", downloadedTarball)
	}
	t.Logf("Tarball location: %s\n", downloadedTarball)

	b, _ := setBuilder(t)

	b.App.Name = "test"
	b.App.Source.URL = "file://" + filepath.Join(downloadDir, tarballFilename)
	if err = b.Load(true); err != nil {
		t.Fatalf("unable to load builder: %s", err)
	}

	res = b.Install()
	if res.Err != nil {
		t.Fatalf("unable to install test tarball: %s", res.Err)
	}
	if b.Env.SrcPath == "" {
		t.Fatal("SrcPath is undefined")
	}

	// Tarball available locally should have been copied into the build directory.
	targetFile := filepath.Join(b.Env.BuildDir, b.App.Name, tarballFilename)
	if !util.FileExists(targetFile) {
		t.Fatalf("expected file %s does not exist", targetFile)
	}
	requireBinaryExists(t, b.Env.InstallDir, b.App.Name, "helloworld")

	expectedTarball := filepath.Join(b.Env.BuildDir, b.App.Name, tarballFilename)
	if b.Env.SrcPath != expectedTarball {
		t.Fatalf("expected tarball is missing: %s instead of %s", b.Env.SrcPath, expectedTarball)
	}
}

// TestUninstall_removesInstallDir verifies that Uninstall() removes the install directory when not in persistent mode.
func TestUninstall_removesInstallDir(t *testing.T) {
	b, _ := setBuilder(t)
	b.App.Name = "c_hello_world"
	b.App.Source.URL = "https://github.com/gvallee/c_hello_world/releases/download/v1.0.1/c_hello_world-1.0.1.tar.gz"
	b.App.Version = "1.0.1"

	if err := b.Load(false); err != nil {
		t.Fatalf("load: %s", err)
	}
	res := b.Install()
	if res.Err != nil {
		t.Fatalf("install: %s", res.Err)
	}
	installDir := b.Env.InstallDir
	if !util.PathExists(installDir) {
		t.Fatal("install dir should exist after Install()")
	}

	res = b.Uninstall()
	if res.Err != nil {
		t.Fatalf("uninstall: %s", res.Err)
	}
	if util.PathExists(installDir) {
		t.Fatalf("install dir %s should have been removed by Uninstall()", installDir)
	}
}

func TestLoad_validation(t *testing.T) {
	t.Run("missing app name", func(t *testing.T) {
		b, _ := setBuilder(t)
		b.App.Source.URL = "https://example.com/foo"
		err := b.Load(false)
		if err == nil {
			t.Fatal("Load expected to fail with empty App.Name")
		}
		if err.Error() != "application's name is undefined" {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("missing URL", func(t *testing.T) {
		b, _ := setBuilder(t)
		b.App.Name = "foo"
		err := b.Load(false)
		if err == nil {
			t.Fatal("Load expected to fail with empty URL")
		}
		if err.Error() != "the URL to download application is undefined" {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("missing scratch dir", func(t *testing.T) {
		b, _ := setBuilder(t)
		b.App.Name = "foo"
		b.App.Source.URL = "https://example.com/foo"
		b.Env.ScratchDir = ""
		err := b.Load(false)
		if err == nil {
			t.Fatal("Load expected to fail with empty ScratchDir")
		}
		if err.Error() != "scratch directory is undefined" {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("missing build dir", func(t *testing.T) {
		b, _ := setBuilder(t)
		b.App.Name = "foo"
		b.App.Source.URL = "https://example.com/foo"
		b.Env.BuildDir = ""
		err := b.Load(false)
		if err == nil {
			t.Fatal("Load expected to fail with empty BuildDir")
		}
		if err.Error() != "build directory is undefined" {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("missing install dir", func(t *testing.T) {
		b, _ := setBuilder(t)
		b.App.Name = "foo"
		b.App.Source.URL = "https://example.com/foo"
		b.Env.InstallDir = ""
		err := b.Load(false)
		if err == nil {
			t.Fatal("Load expected to fail with empty InstallDir")
		}
		if err.Error() != "install directory is undefined" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestUninstall_persistentMode_noOp(t *testing.T) {
	b, _ := setBuilder(t)
	b.Persistent = b.Env.InstallDir // non-empty so Uninstall does nothing
	b.App.Name = "c_hello_world"
	b.App.Source.URL = "https://github.com/gvallee/c_hello_world/releases/download/v1.0.1/c_hello_world-1.0.1.tar.gz"

	if err := b.Load(false); err != nil {
		t.Fatalf("load: %s", err)
	}
	res := b.Install()
	if res.Err != nil {
		t.Fatalf("install: %s", res.Err)
	}
	installDir := b.Env.InstallDir
	if !util.PathExists(installDir) {
		t.Fatal("install dir should exist after Install()")
	}

	res = b.Uninstall()
	if res.Err != nil {
		t.Fatalf("uninstall: %s", res.Err)
	}
	if !util.PathExists(installDir) {
		t.Fatal("in persistent mode Uninstall() should not remove the install dir")
	}
}

func TestCompile_failsWhenGetFails(t *testing.T) {
	b, _ := setBuilder(t)
	b.App.Name = "foo"
	b.App.Source.URL = "" // invalid: Get() will fail

	err := b.Compile()
	if err == nil {
		t.Fatal("Compile expected to fail when URL is empty")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}
