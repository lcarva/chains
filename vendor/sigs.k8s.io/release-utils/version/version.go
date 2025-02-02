/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"text/tabwriter"

	"github.com/common-nighthawk/go-figure"
)

// Base version information.
//
// This is the fallback data used when version information from git is not
// provided via go ldflags.
var (
	// Output of "git describe". The prerequisite is that the
	// branch should be tagged using the correct versioning strategy.
	gitVersion = "devel"
	// SHA1 from git, output of $(git rev-parse HEAD)
	gitCommit = "unknown"
	// State of git tree, either "clean" or "dirty"
	gitTreeState = "unknown"
	// Build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
	buildDate = "unknown"
	// flag to print the ascii name banner
	asciiName = "true"
)

type Info struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`

	ASCIIName   string `json:"-"`
	Name        string `json:"-"`
	Description string `json:"-"`
}

// GetVersionInfo represents known information on how this binary was built.
func GetVersionInfo() Info {
	info := Info{
		ASCIIName:    asciiName,
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	// Look for the default version and replace it from runtime build info if possible.
	if info.GitVersion != "devel" {
		return info
	}

	// If there is debug info for the module, this binary was installed outside
	// the normal build process and might not have the ld flags set.
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	// Version is set in artifacts built with -X sigs.k8s.io/release-utils/version.gitVersion=<version>
	// Ensure version is also set when installed via go install <module>
	info.GitVersion = bi.Main.Version

	return info
}

// String returns the string representation of the version info
func (i *Info) String() string {
	b := strings.Builder{}
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)

	// name and description are optional.
	if i.Name != "" {
		if i.ASCIIName == "true" {
			f := figure.NewFigure(strings.ToUpper(i.Name), "", true)
			_, _ = fmt.Fprint(w, f.String())
		}
		_, _ = fmt.Fprint(w, i.Name)
		if i.Description != "" {
			_, _ = fmt.Fprintf(w, ": %s", i.Description)
		}
		_, _ = fmt.Fprint(w, "\n\n")
	}

	_, _ = fmt.Fprintf(w, "GitVersion:\t%s\n", i.GitVersion)
	_, _ = fmt.Fprintf(w, "GitCommit:\t%s\n", i.GitCommit)
	_, _ = fmt.Fprintf(w, "GitTreeState:\t%s\n", i.GitTreeState)
	_, _ = fmt.Fprintf(w, "BuildDate:\t%s\n", i.BuildDate)
	_, _ = fmt.Fprintf(w, "GoVersion:\t%s\n", i.GoVersion)
	_, _ = fmt.Fprintf(w, "Compiler:\t%s\n", i.Compiler)
	_, _ = fmt.Fprintf(w, "Platform:\t%s\n", i.Platform)

	_ = w.Flush()
	return b.String()
}

// JSONString returns the JSON representation of the version info
func (i *Info) JSONString() (string, error) {
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}
