// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package builder

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gvallee/go_util/pkg/util"
)

func setBuilder(t *testing.T) *Builder {
	b := new(Builder)

	var err error
	b.Env.ScratchDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create scratch directory: %s", err)
	}
	defer os.RemoveAll(b.Env.ScratchDir)
	t.Logf("Scratch directory: %s", b.Env.ScratchDir)

	b.Env.InstallDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}
	t.Logf("Install directory: %s", b.Env.InstallDir)

	b.Env.BuildDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create build directory: %s", err)
	}
	t.Logf("Build directory: %s", b.Env.BuildDir)

	return b
}

func TestInstallFromAutotoolsRelease(t *testing.T) {
	b := setBuilder(t)
	defer os.RemoveAll(b.Env.InstallDir)
	defer os.RemoveAll(b.Env.BuildDir)
	t.Logf("Build directory: %s", b.Env.BuildDir)
	t.Logf("Source directory: %s", b.Env.SrcDir)
	b.App.Name = "c_hello_world"
	b.App.URL = "https://github.com/gvallee/c_hello_world/releases/download/v1.0.1/c_hello_world-1.0.1.tar.gz"
	b.App.Version = "1.0.1"

	err := b.Load(false)
	if err != nil {
		t.Fatalf("unable to load the builder: %s", err)
	}

	if b.Configure == nil {
		t.Fatalf("builder configure is undefined")
	}

	res := b.Install()
	if res.Err != nil {
		t.Fatalf("unable to install the software package: %s", res.Err)
	}

	expectedBinary := filepath.Join(b.Env.InstallDir, b.App.Name, "bin", "helloworld")
	if !util.FileExists(expectedBinary) {
		t.Fatalf("expected binary %s does not exist", expectedBinary)
	}

}

func TestInstallFromSource(t *testing.T) {
	b := setBuilder(t)
	t.Logf("Build directory: %s", b.Env.BuildDir)
	t.Logf("Source directory: %s", b.Env.SrcDir)
	b.App.Name = "helloworld"
	b.App.URL = "https://github.com/gvallee/c_hello_world/archive/1.0.0.tar.gz"
	b.App.Version = "1.0.0"

	err := b.Load(false)
	if err != nil {
		t.Fatalf("unable to load the builder: %s", err)
	}

	if b.Configure == nil {
		t.Fatalf("builder configure is undefined")
	}

	res := b.Install()
	if res.Err != nil {
		t.Fatalf("unable to install the software package: %s", res.Err)
	}

	expectedBinary := filepath.Join(b.Env.InstallDir, b.App.Name, "bin", "helloworld")
	if !util.FileExists(expectedBinary) {
		t.Fatalf("expected binary %s does not exist", expectedBinary)
	}
}
