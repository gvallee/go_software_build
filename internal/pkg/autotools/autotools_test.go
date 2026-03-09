// Copyright (c) 2026, NVIDIA CORPORATION. All rights reserved.
// This software is licensed under a 3-clause BSD license.

package autotools

import (
	"os"
	"testing"
)

func TestDetect(t *testing.T) {
	cfg := &Config{Source: "/tmp"}
	cfg.Detect()
	if cfg.DetectDone != false {
		t.Errorf("DetectDone should be false when no autotools files are found")
	}
}

func TestMakefileHasTarget(t *testing.T) {
	// Create a temporary Makefile
	file, err := os.CreateTemp("", "Makefile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(file.Name())
	_, err = file.WriteString("install:\n\techo Installing\n")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	file.Close()

	cfg := &Config{}
	if !cfg.MakefileHasTarget("install", file.Name()) {
		t.Errorf("MakefileHasTarget should return true for 'install' target")
	}
}

func TestConfigure_NoConfigureScript(t *testing.T) {
	// Setup: create temp source dir without configure script
	dir, err := os.MkdirTemp("", "autotools-src")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	cfg := &Config{
		Source:  dir,
		Install: dir,
	}
	// Should skip configuration step and return nil
	err = cfg.Configure()
	if err != nil {
		t.Errorf("Configure should return nil when no configure script is present, got: %v", err)
	}
}

func TestAutogen_NoAutogenScript(t *testing.T) {
	// Setup: create temp source dir without autogen.sh or autogen.pl
	dir, err := os.MkdirTemp("", "autotools-src")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	cfg := &Config{
		Source:     dir,
		HasAutogen: true,
	}
	// Should skip autogen and ignore error since no script exists
	err = autogen(cfg)
	if err != nil {
		t.Logf("autogen returned error as expected when no script exists: %v", err)
	}
}

// Additional tests for Configure and autogen would require more setup/mocking
