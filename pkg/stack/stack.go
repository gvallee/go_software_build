//
// Copyright (c) 2023, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package stack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gvallee/go_software_build/internal/pkg/module"
	"github.com/gvallee/go_software_build/pkg/builder"
	"github.com/gvallee/go_util/pkg/util"
)

type StackCfg struct {
	InstallDir string `json:"installDir"`
	System     string `json:"system"`
}

type Component struct {
	// Name of the software component, e.g., 'ompi'
	Name string `json:"name"`

	// URL to use to get the software component
	URL string `json:"URL"`

	// When applicable, i.e., with Git, which branch to use when getting the software component
	Branch string `json:"branch"`

	// Command to execute before getting the code from a branch. Can be used to get all the tags of a Git repository.
	BranchCheckoutPrelude string `json:"branch_checkout_prelude"`

	// ConfigId presents the configure option to use by other components with a dependency, e.g., will result in `--with-<ConfigID>` when autotools end up being used
	ConfigId string `json:"configure_id"`

	// Dependency for the software component, must be the name of another component
	ConfigureDependency string `json:"configure_dependency"`

	// ConfigurePrelude is the command to execute before configuring the software component. Can be used to initialize Git submodules for example.
	ConfigurePrelude string `json:"configure_prelude"`

	// ConfigureParams represents the additional configure parameters
	ConfigureParams string `json:"configure_params"`
}

type StackDef struct {
	// Name of the stack
	Name string `json:"name"`

	// System targeted by the stack (e.g., host, dpu)
	System string `json:"system"`

	// Type of the stack, i.e., private or public
	Type string `json:"type"`

	// Components represents the list of components composing the stack
	Components []Component `json:"components"`
}

type Stack struct {
	Private         bool
	BuildEnv        []string
	StackConfig     *StackCfg
	StackDefinition *StackDef
}

type Config struct {
	// DefFilePath is the path to the file defining the stack
	DefFilePath string

	// ConfigFilePath is the path to the file specifying the configuration of the stack
	ConfigFilePath string

	// Loaded specifies is the stack configuration is ready to be used or not, either through manual setting or parsing of configuration files.
	Loaded bool

	// Data represents all the details about the stack, including the data from the configuration files once they are parsed
	Data Stack

	// Map of all software components installed for the stack. The key is the name of the component and the value the directory where the component is installed
	InstalledComponents map[string]string
}

const (
	defaultPermission = 0775
)

func getComponentsPathEnv(env []string, cfg *Config) []string {
	var compsEnv []string
	if env != nil {
		compsEnv = append(compsEnv, env...)
	}
	for _, compDir := range cfg.InstalledComponents {
		compBinPath := filepath.Join(compDir, "bin")
		if util.PathExists(compBinPath) {
			compsEnv = append(compsEnv, compBinPath)
			fmt.Printf("[DBG] %s added to path\n", compBinPath)
		}
	}
	return compsEnv
}

func (c *Config) Load() error {
	// unmarshale the two configuration files
	defFile, err := os.Open(c.DefFilePath)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", c.DefFilePath, err)
	}
	defContent, err := ioutil.ReadAll(defFile)
	if err != nil {
		return fmt.Errorf("unable to read the content of %s: %w", c.DefFilePath, err)
	}
	c.Data.StackDefinition = new(StackDef)
	err = json.Unmarshal(defContent, &c.Data.StackDefinition)
	if err != nil {
		return fmt.Errorf("unable to unmarshal content of %s: %w", c.DefFilePath, err)
	}

	cfgFile, err := os.Open(c.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", c.ConfigFilePath, err)
	}
	cfgContent, err := ioutil.ReadAll(cfgFile)
	if err != nil {
		return fmt.Errorf("unable to read the content of %s: %w", c.ConfigFilePath, err)
	}
	c.Data.StackConfig = new(StackCfg)
	err = json.Unmarshal(cfgContent, &c.Data.StackConfig)
	if err != nil {
		return fmt.Errorf("unable to unmarshal content of %s: %w", c.ConfigFilePath, err)
	}
	c.Loaded = true
	return nil
}

func createNewPathForComp(compBinDir string) string {
	existingPath := os.Getenv("PATH")
	return "PATH="+compBinDir+":"+existingPath+":$PATH"
}

func (c *Config) InstallStack() error {
	// A map of all the installed components where the key of the component's name and the value the directory where it is installed
	installedComponents := make(map[string]string)

	// A map of all the identifiers used to configure the different components with dependencies
	configIds := make(map[string]string)

	if !c.Loaded {
		err := c.Load()
		if err != nil {
			return fmt.Errorf("unable to load configuration: %w", err)
		}
	}

	if c.Data.StackDefinition.Type == "private" && c.Data.Private {
		return fmt.Errorf("you are trying to install a private stack on a public system, which is strictly prohibited! Please use the -private option if you are on a private system to deploy the target stack")
	}

	if !util.PathExists(c.Data.StackConfig.InstallDir) {
		err := os.MkdirAll(c.Data.StackConfig.InstallDir, defaultPermission)
		if err != nil {
			return fmt.Errorf("unable to create installation directory %s: %w", c.Data.StackConfig.InstallDir, err)
		}
	}

	for _, softwareComponents := range c.Data.StackDefinition.Components {
		// Set a builder
		b := new(builder.Builder)

		stackBasedir := filepath.Join(c.Data.StackConfig.InstallDir, c.Data.StackDefinition.Name)
		if !util.PathExists(stackBasedir) {
			err := os.MkdirAll(stackBasedir, defaultPermission)
			if err != nil {
				return fmt.Errorf("unable to create %s: %w", stackBasedir, err)
			}
		}
		b.Env.ScratchDir = filepath.Join(stackBasedir, "scratch")
		b.Env.InstallDir = filepath.Join(stackBasedir, "install")
		b.Env.BuildDir = filepath.Join(stackBasedir, "build")
		b.Env.SrcDir = filepath.Join(stackBasedir, "src")
		b.Env.Env = c.Data.BuildEnv

		if !util.PathExists(b.Env.ScratchDir) {
			err := os.MkdirAll(b.Env.ScratchDir, defaultPermission)
			if err != nil {
				return fmt.Errorf("unable to create %s: %w", b.Env.ScratchDir, err)
			}
		}

		if !util.PathExists(b.Env.InstallDir) {
			err := os.MkdirAll(b.Env.InstallDir, defaultPermission)
			if err != nil {
				return fmt.Errorf("unable to create %s: %w", b.Env.InstallDir, err)
			}
		}

		if !util.PathExists(b.Env.BuildDir) {
			err := os.MkdirAll(b.Env.BuildDir, defaultPermission)
			if err != nil {
				return fmt.Errorf("unable to create %s: %w", b.Env.BuildDir, err)
			}
		}

		if !util.PathExists(b.Env.SrcDir) {
			err := os.MkdirAll(b.Env.SrcDir, defaultPermission)
			if err != nil {
				return fmt.Errorf("unable to create %s: %w", b.Env.SrcDir, err)
			}
		}

		log.Printf("-> Installing %s", softwareComponents.Name)
		b.App.Name = softwareComponents.Name
		b.App.Source.URL = softwareComponents.URL
		b.App.Source.Branch = softwareComponents.Branch

		if softwareComponents.ConfigureDependency != "" {
			deps := strings.Split(softwareComponents.ConfigureDependency, ",")
			for _, dep := range deps {
				ref := dep
				_, ok := configIds[dep]
				if ok {
					ref = configIds[dep]
				}
				configureOption := fmt.Sprintf("--with-%s=%s", ref, installedComponents[dep])
				b.App.AutotoolsCfg.ExtraConfigureArgs = append(b.App.AutotoolsCfg.ExtraConfigureArgs, configureOption)
			}
		}

		if softwareComponents.ConfigureParams != "" {
			args := strings.Split(softwareComponents.ConfigureParams, " ")
			b.App.AutotoolsCfg.ExtraConfigureArgs = append(b.App.AutotoolsCfg.ExtraConfigureArgs, args...)
		}

		if softwareComponents.ConfigurePrelude != "" {
			b.App.AutotoolsCfg.ConfigurePreludeCmd = softwareComponents.ConfigurePrelude
		}

		if softwareComponents.BranchCheckoutPrelude != "" {
			b.App.Source.BranchCheckoutPrelude = softwareComponents.BranchCheckoutPrelude
		}

		err := b.Load(true)
		if err != nil {
			return fmt.Errorf("unable to load the builder for %s: %w", b.App.Name, err)
		}

		res := b.Install()
		if res.Err != nil {
			return fmt.Errorf("unable to install %s: %w", softwareComponents.Name, res.Err)
		}

		if softwareComponents.ConfigId != "" {
			configIds[softwareComponents.Name] = softwareComponents.ConfigId
		}

		// Track what was installed, both locally and globally
		compInstallDir := filepath.Join(b.Env.InstallDir, softwareComponents.Name)
		installedComponents[softwareComponents.Name] = compInstallDir
		if c.InstalledComponents == nil {
			c.InstalledComponents = make(map[string]string)
		}
		c.InstalledComponents[softwareComponents.Name] = compInstallDir

		compBinDir := filepath.Join(compInstallDir, "bin")
		if util.PathExists(compBinDir) {
			fmt.Printf("[DBG] Adding %s to PATH\n", compBinDir)
		}

		if len(c.Data.BuildEnv) == 0 {
			// Create a new environment
			pathEnvStr := createNewPathForComp(compBinDir)
			c.Data.BuildEnv = append(c.Data.BuildEnv, pathEnvStr)
		} else {
			// Do we have a PATH env already?
			pathFound := false
			for idx, envvar := range c.Data.BuildEnv {
				tokens := strings.Split(envvar, "=")
				if tokens[0] == "PATH" {
					newPath := "PATH="+compBinDir+":"+tokens[1]
					c.Data.BuildEnv[idx] = newPath
					pathFound = true
					break
				}
			}
			if pathFound == false {
				// We can set a new one
				pathEnvStr := createNewPathForComp(compBinDir)
				c.Data.BuildEnv = append(c.Data.BuildEnv, pathEnvStr)
			}
		}
		fmt.Printf("[DBG] Env: %s\n", strings.Join(c.Data.BuildEnv, "\n"))

		log.Printf("-> %s was successfully installed in %s", softwareComponents.Name, compInstallDir)
	}

	return nil
}

func (c *Config) Export() error {
	err := c.Load()
	if err != nil {
		return fmt.Errorf("c.Load() failed: %w", err)
	}

	stackBasedir := filepath.Join(c.Data.StackConfig.InstallDir, c.Data.StackDefinition.Name)
	if !util.PathExists(stackBasedir) {
		return fmt.Errorf("%s does not exist", stackBasedir)
	}

	installDir := filepath.Join(stackBasedir, "install")
	if !util.PathExists(installDir) {
		return fmt.Errorf("%s does not exist", installDir)
	}

	tarballFilename := c.Data.StackDefinition.Name + ".tar.bz2"
	tarBin, err := exec.LookPath("tar")
	if err != nil {
		return fmt.Errorf("tar is not available: %w", err)
	}
	tarCmd := exec.Command(tarBin, "-cjf", tarballFilename, "install")
	tarCmd.Dir = stackBasedir
	var stderr, stdout bytes.Buffer
	tarCmd.Stderr = &stderr
	tarCmd.Stdout = &stdout
	err = tarCmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %w - stdout: %s - stderr: %s", err, stdout.String(), stderr.String())
	}

	fmt.Printf("Stack successfully export: %s\n", filepath.Join(stackBasedir, tarballFilename))
	return nil
}

func (c *Config) Import(filePath string) error {
	err := c.Load()
	if err != nil {
		return fmt.Errorf("c.Load() failed: %w", err)
	}

	stackBasedir := filepath.Join(c.Data.StackConfig.InstallDir, c.Data.StackDefinition.Name)
	if !util.PathExists(stackBasedir) {
		err := os.MkdirAll(stackBasedir, defaultPermission)
		if err != nil {
			return fmt.Errorf("unable to create %s: %w", stackBasedir, err)
		}
	}

	tarBin, err := exec.LookPath("tar")
	if err != nil {
		return fmt.Errorf("ERROR: tar is not available: %w", err)
	}
	tarCmd := exec.Command(tarBin, "-xjf", filePath)
	tarCmd.Dir = stackBasedir
	var stderr, stdout bytes.Buffer
	tarCmd.Stderr = &stderr
	tarCmd.Stdout = &stdout
	err = tarCmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %s - stdout: %s - stderr: %s", err, stdout.String(), stderr.String())
	}

	fmt.Printf("Stack successfully import in %s\n", stackBasedir)
	return nil
}

func (c *Config) GenerateModules(copyright, customEnvVarPrefix string) error {
	err := c.Load()
	if err != nil {
		return fmt.Errorf("c.Load() failed: %w", err)
	}

	stackBasedir := filepath.Join(c.Data.StackConfig.InstallDir, c.Data.StackDefinition.Name)
	if !util.PathExists(stackBasedir) {
		return fmt.Errorf("stack base directory %s does not exist", stackBasedir)
	}

	modulefileDir := filepath.Join(stackBasedir, "modulefiles")
	if !util.PathExists(modulefileDir) {
		err := os.MkdirAll(modulefileDir, defaultPermission)
		if err != nil {
			return fmt.Errorf("unable to create %s: %w", modulefileDir, err)
		}
	}

	for _, softwareComponent := range c.Data.StackDefinition.Components {
		var requires []string
		vars := make(map[string]string)
		envVars := make(map[string]string)
		envLayout := make(map[string][]string)

		// Set the requirements
		if softwareComponent.ConfigureDependency != "" {
			deps := strings.Split(softwareComponent.ConfigureDependency, ",")
			requires = append(requires, deps...)
		}

		// Set the vars
		vars["software_stack_dir"] = stackBasedir

		compInstallDir := filepath.Join(stackBasedir, "install", softwareComponent.Name)
		compBinDir := filepath.Join(compInstallDir, "bin")
		compLibDir := filepath.Join(compInstallDir, "lib")
		compIncDir := filepath.Join(compInstallDir, "include")
		compManDir := filepath.Join(compInstallDir, "man")
		compPkgDir := filepath.Join(compLibDir, "pkgconfig")

		// Set the new environment variables
		compBasedirVarName := strings.ToUpper(softwareComponent.Name) + "_DIR"
		compBasedirVarValue := compInstallDir
		envVars[compBasedirVarName] = compBasedirVarValue

		compBuildDir := filepath.Join(stackBasedir, "build", softwareComponent.Name)
		if util.PathExists(compBuildDir) {
			// Figure out the actual directory that was used
			list, err := ioutil.ReadDir(compBuildDir)
			targetDir := ""
			if err != nil {
				return fmt.Errorf("unable to read content of %s: %w", compBuildDir, err)
			}
			for _, entry := range list {
				if util.IsDir(filepath.Join(compBuildDir, entry.Name())) {
					targetDir = entry.Name()
					break
				}
			}
			if targetDir == "" {
				return fmt.Errorf("unable to find build directory from %s: %w", compBuildDir, err)
			}
			compBuildDirVarName := strings.ToUpper(softwareComponent.Name) + "_BUILD_DIR"
			compBuildDirVarValue := filepath.Join(compBuildDir, targetDir)
			envVars[compBuildDirVarName] = compBuildDirVarValue
		} else {
			compSrcDir := filepath.Join(stackBasedir, "src")
			targetDir := ""
			listSrcDirs, err := ioutil.ReadDir(compSrcDir)
			if err == nil {
				// The stack may not have a source directory, for instance when the stack is imported
				// rather than build locally.
				// If the source directory does exist, we set some optional additional environment
				// variables.
				for _, entry := range listSrcDirs {
					if strings.Contains(entry.Name(), softwareComponent.Name) {
						targetDir = entry.Name()
					}
				}
				if targetDir == "" {
					return fmt.Errorf("unable to find build directory from %s: %w", compSrcDir, err)
				}
				compSrcDirVarName := strings.ToUpper(softwareComponent.Name) + "_BUILD_DIR"
				compSrcDirVarValue := filepath.Join(compSrcDir, targetDir)
				envVars[compSrcDirVarName] = compSrcDirVarValue
			}
		}

		// Prepend existing environment variables
		if util.PathExists(compBinDir) {
			envLayout["PATH"] = append(envLayout["PATH"], compBinDir)
		}

		if util.PathExists(compLibDir) {
			envLayout["LIBRARY_PATH"] = append(envLayout["LIBRARY_PATH"], compLibDir)
			envLayout["LD_LIBRARY_PATH"] = append(envLayout["LD_LIBRARY_PATH"], compLibDir)
		}

		if util.PathExists(compIncDir) {
			envLayout["CPATH"] = append(envLayout["CPATH"], compIncDir)
		}

		if util.PathExists(compManDir) {
			envLayout["MANPATH"] = append(envLayout["MANPATH"], compManDir)
		}

		if util.PathExists(compPkgDir) {
			envLayout["PKG_CONFIG_PATH"] = append(envLayout["PKG_CONFIG_PATH"], compPkgDir)
		}

		err := module.Generate(modulefileDir, copyright, customEnvVarPrefix, softwareComponent.Name, requires, nil, vars, envVars, envLayout)
		if err != nil {
			return fmt.Errorf("module.Generate() failed: %w", err)
		}
	}

	fmt.Printf("modules successfully creates, to use them: module use %s\n", modulefileDir)
	return nil
}
