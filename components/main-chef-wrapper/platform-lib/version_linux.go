//
// Copyright (c) Chef Software, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// logic for linux platform
package platform_lib

import (
	"encoding/json"
	"fmt"
	"github.com/chef/chef-workstation/components/main-chef-wrapper/dist"
	"github.com/chef/chef-workstation/components/main-chef-wrapper/lib"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var gemManifestMap map[string]interface{}
var manifestMap map[string]interface{}

func GlobalReadFile() {
	gemManifestMap = gemManifestHash()
	manifestMap = manifestHash()
}
func Version() error {
	if OmnibusInstall() {
		showVersionViaVersionManifest()
	} else {
		fmt.Fprintln(os.Stderr, "ERROR:", dist.WorkstationProduct, "has not been installed via the platform-specific package provided by", dist.DistributorName, "Version information is not available.")

	}
	return nil
}

func showVersionViaVersionManifest() {
	fmt.Printf("%v version: %v", dist.WorkstationProduct, componentVersion("build_version"))
	productMap := map[string]string{
		dist.ClientProduct: dist.ClientGem,
		dist.InspecProduct: dist.InspecGem,
		dist.CliProduct:    dist.CliGem,
		dist.HabProduct:    dist.HabSoftwareName,
		"Test Kitchen":     "test-kitchen",
		"Cookstyle":        "cookstyle",
	}
	for prodName, component := range productMap {
		fmt.Printf("\n%v version: %v", prodName, componentVersion(component))
	}
	fmt.Printf("\n")
}
func componentVersion(component string) string {
	v, ok := gemManifestMap[component]
	if ok {
		stringifyVal := v.([]interface{})[0]
		return stringifyVal.(string)
	} else if v, ok := manifestMap[component]; ok {
		return v.(string)
	} else {
		success, _ := lib.Dig(manifestMap, "software", component, "locked_version")
		if success == nil {
			return "unknown"
		} else {
			return success.(string)
		}
	}
}
func gemManifestHash() map[string]interface{} {
	filepath := path.Join(omnibusRoot(), "gem-version-manifest.json")
	jsonFile, err := os.Open(filepath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err.Error())
		os.Exit(4)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var gemManifestHash map[string]interface{}
	json.Unmarshal([]byte(byteValue), &gemManifestHash)
	defer jsonFile.Close()
	return gemManifestHash
}
func manifestHash() map[string]interface{} {
	filepath := path.Join(omnibusRoot(), "version-manifest.json")
	jsonFile, err := os.Open(filepath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err.Error())
		os.Exit(4)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var manifestHash map[string]interface{}
	json.Unmarshal([]byte(byteValue), &manifestHash)
	defer jsonFile.Close()
	return manifestHash
}

func OmnibusInstall() bool {
	//# We also check if the location we're running from (omnibus_root is relative to currently-running ruby)
	//# includes the version manifest that omnibus packages ship with. If it doesn't, then we're running locally
	//# or out of a gem - so not as an 'omnibus install'
	ExpectedOmnibusRoot := ExpectedOmnibusRoot()
	if _, err := os.Stat(ExpectedOmnibusRoot); err == nil {
		if _, err = os.Stat(path.Join(ExpectedOmnibusRoot, "version-manifest.json")); err == nil {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func omnibusRoot() string {
	omnibusroot, err := filepath.Abs(path.Join(ExpectedOmnibusRoot()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", dist.WorkstationProduct, "has not been installed via the platform-specific package provided by", dist.DistributorName, "Version information is not available.")
		os.Exit(4)
	}
	return omnibusroot
	//below code can be used for running and testing in local repos e.g ./main-chef-wrapper -v, comment out rest code of this method(darwin,linux)
	//return "/opt/chef-workstation"
}

func ExpectedOmnibusRoot() string {
	ex, _ := os.Executable()
	exReal, err := filepath.EvalSymlinks(ex)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(4)
	}
	rootPath := path.Join(filepath.Dir(exReal), "..")
	return rootPath
	//below code can be used for running and testing in local repos e.g ./main-chef-wrapper -v, comment out rest code of this method(darwin,linux)
	//return "/opt/chef-workstation"
}

func UnmarshallRubyEnv() map[string]interface{} {
	home, err := os.UserHomeDir()
	filepath := path.Join(home, ".chef/ruby-env.json")
	jsonFile, err := os.Open(filepath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err.Error())
		return nil
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	defer jsonFile.Close()
	var rubyenvHash map[string]interface{}
	json.Unmarshal([]byte(byteValue), &rubyenvHash)
	return rubyenvHash
}

func MatchVersions() bool {
	// check version from env.json file and workstation version
	WorkstationVersion, envDoc, b, done := EnvDoc()
	if done {
		return b
	}
	if envDoc["build_version"] == WorkstationVersion {
		return true
	} else {
		return false
	}
}

func EnvDoc() (string, map[string]interface{}, bool, bool) {
	home, err := os.UserHomeDir()
	WorkstationVersion := componentVersion("build_version")
	filepath := path.Join(home, ".chef/ruby-env.json")
	jsonFile, err := os.Open(filepath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err.Error())
		return "", nil, false, true
	}

	data, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
		return "", nil, false, true
	}

	envDoc := make(map[string]interface{})
	if err := json.Unmarshal(data, &envDoc); err != nil {
		log.Fatal(err)
		return "", nil, false, true
	}
	defer jsonFile.Close()
	return WorkstationVersion, envDoc, false, false
}

func DefaultChefRuby() bool {
	absPath, b, done := AbsoluteRubyPath()
	if done {
		return b
	}
	if absPath == "/opt/chef-workstation/embedded/bin/ruby" {
		return true
	} else {
		return false
	}
}

func AbsoluteRubyPath() (string, bool, bool) {
	out, err := exec.Command("which", "ruby").Output()
	if err != nil {
		return "", false, true
	}
	stringOut := string(out)
	Sanatizepath := strings.Replace(stringOut, "\n", "", -1)
	absPath, _ := filepath.Abs(Sanatizepath)
	return absPath, false, false
}
