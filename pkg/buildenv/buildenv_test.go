// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildenv

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	appInfo.URL = "file://" + dir1

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
	t.Logf("Checking if %s was correctly installed,,,", expectedFile)
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
	a.URL = "https://github.com/gvallee/c_hello_world/archive/1.0.0.tar.gz"
	a.Version = "1.0.0"

	testEnv := new(Info)
	var err error
	testEnv.BuildDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %s", err)
	}

	err = testEnv.Get(a)
	if err != nil {
		t.Fatalf("unable to download %s: %s", a.URL, err)
	}

	return testEnv, a
}

func TestFileURLGet(t *testing.T) {
	testEnv, a := getHelloWorldSrc(t)
	defer os.RemoveAll(testEnv.BuildDir)

	var expectedEnv Info
	expectedEnv.SrcDir = filepath.Join(testEnv.BuildDir, a.Name)
	expectedEnv.SrcPath = filepath.Join(expectedEnv.SrcDir, filepath.Base(a.URL))
	t.Logf("Checking if %s was correctly installed,,,", expectedEnv.SrcPath)
	if !util.FileExists(expectedEnv.SrcPath) {
		t.Fatalf("expected file %s does not exist", expectedEnv.SrcPath)
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
				URL:  "https://something/toto.tar.gz",
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
				URL:  "https://something/application2.tar.bz2",
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
				URL:  "https://something/application3.git",
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
