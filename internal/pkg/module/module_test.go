package module

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate_isDeterministic(t *testing.T) {
	dir, err := ioutil.TempDir("", "module_gen_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	varsA := map[string]string{"B_VAR": "2", "A_VAR": "1"}
	envVarsA := map[string]string{"B_ENV": "b", "A_ENV": "a"}
	envLayoutA := map[string][]string{"LD_LIBRARY_PATH": {"/lib"}, "PATH": {"/bin"}}

	if err := Generate(dir, "copyright", "", "mymodule", []string{"dep2", "dep1"}, []string{"conflict1"}, varsA, envVarsA, envLayoutA); err != nil {
		t.Fatalf("Generate first: %v", err)
	}
	firstPath := filepath.Join(dir, "mymodule")
	firstContent, err := ioutil.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("ReadFile first: %v", err)
	}

	varsB := map[string]string{"A_VAR": "1", "B_VAR": "2"}
	envVarsB := map[string]string{"A_ENV": "a", "B_ENV": "b"}
	envLayoutB := map[string][]string{"PATH": {"/bin"}, "LD_LIBRARY_PATH": {"/lib"}}

	if err := Generate(dir, "copyright", "", "mymodule", []string{"dep2", "dep1"}, []string{"conflict1"}, varsB, envVarsB, envLayoutB); err != nil {
		t.Fatalf("Generate second: %v", err)
	}
	secondContent, err := ioutil.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("ReadFile second: %v", err)
	}

	if string(firstContent) != string(secondContent) {
		t.Fatal("Generate output is not deterministic for equivalent map content")
	}

	contentStr := string(secondContent)
	if strings.Index(contentStr, "set A_VAR 1\n") > strings.Index(contentStr, "set B_VAR 2\n") {
		t.Fatal("expected sorted 'set' entries")
	}
	if strings.Index(contentStr, "setenv A_ENV a\n") > strings.Index(contentStr, "setenv B_ENV b\n") {
		t.Fatal("expected sorted 'setenv' entries")
	}
	if strings.Index(contentStr, "prepend-path LD_LIBRARY_PATH /lib\n") > strings.Index(contentStr, "prepend-path PATH /bin\n") {
		t.Fatal("expected sorted 'prepend-path' entries")
	}
}

func TestGenerate_customEnvVarPrefix(t *testing.T) {
	dir, err := ioutil.TempDir("", "module_prefix_")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	envVars := map[string]string{
		"PATH":            "/custom/bin",
		"MYP_FOO":         "foo",
		"LD_LIBRARY_PATH": "/custom/lib",
	}

	if err := Generate(dir, "copyright", "MYP_", "prefmodule", nil, nil, nil, envVars, nil); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	content, err := ioutil.ReadFile(filepath.Join(dir, "prefmodule"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "setenv MYP_PATH /custom/bin\n") {
		t.Fatal("expected prefixed PATH variable")
	}
	if !strings.Contains(contentStr, "setenv MYP_FOO foo\n") {
		t.Fatal("expected existing prefixed variable to be preserved")
	}
	if !strings.Contains(contentStr, "setenv MYP_LD_LIBRARY_PATH /custom/lib\n") {
		t.Fatal("expected prefixed LD_LIBRARY_PATH variable")
	}
}
