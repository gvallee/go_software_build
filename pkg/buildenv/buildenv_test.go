// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildenv

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gvallee/go_software_build/pkg/app"
	"github.com/gvallee/go_util/pkg/util"
)

func TestDirURLGet(t *testing.T) {
	var testEnv Info
	var appInfo app.Info

	// The get operation is expected to result with a copy an entire directory into another directory
	dir1, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(dir1)
	appInfo.Name = "testApp"
	appInfo.URL = "file://"+dir1

	tempFile, err := ioutil.TempFile(dir1, "")
	if err != nil {
		t.Fatalf("unable to create temporary file: %s", err)
	}

	testEnv.BuildDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(testEnv.BuildDir)

	err = testEnv.Get(&appInfo) 
	if err != nil {
		t.Fatalf("Get() failed: %s", err)
	}

	expectedDir := filepath.Join(testEnv.BuildDir, appInfo.Name, filepath.Base(dir1))
	expectedFile := filepath.Join(expectedDir, filepath.Base(tempFile.Name()))
	if !util.FileExists(expectedFile) {
		t.Fatalf("expected file %s does not exist", expectedFile)
	}
	
	if testEnv.SrcPath != expectedDir {
		t.Fatalf("source path has not been properly set (%s instead of %s)", testEnv.SrcPath, expectedDir)
	}
}

func TestFileURLGet(t *testing.T) {
	var a app.Info
	a.Name = "helloworld"
	a.URL = "https://github.com/gvallee/c_hello_world/archive/1.0.0.tar.gz"
	a.Version = "1.0.0"

	var testEnv Info
	var err error
	testEnv.SrcDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(testEnv.BuildDir)

	err = testEnv.Get(&a)
	if err != nil {
		t.Fatalf("unable to download %s: %s", a.URL, err)
	}

	expectedFile := filepath.Join(testEnv.SrcDir, a.Name, filepath.Base(a.URL))
	if !util.FileExists(expectedFile) {
		t.Fatalf("expected file %s does not exist", expectedFile)
	}

	if testEnv.SrcPath != expectedFile {
		t.Fatalf("source path has not been properly set (%s instead of %s)", testEnv.SrcPath, expectedFile)
	}
}