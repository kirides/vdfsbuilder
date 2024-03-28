package main

import (
	"strings"
	"time"

	"github.com/kirides/vdfsbuilder"
	"github.com/kirides/vdfsbuilder/vdf"

	"github.com/sethvargo/go-githubactions"
)

func main() {
	inFile := strings.TrimSpace(githubactions.GetInput("in"))
	outFile := strings.TrimSpace(githubactions.GetInput("out"))
	baseDir := strings.TrimSpace(githubactions.GetInput("baseDir"))
	tsOverrideStr := strings.TrimSpace(githubactions.GetInput("ts"))

	vm, err := vdf.ParseVM(inFile)
	if err != nil {
		githubactions.Fatalf("failed to parse input file %q. %v", inFile, err)
	}
	// allow for custom base directory
	if baseDir != "" {
		vm.BaseDir = baseDir
		githubactions.Infof("Overwriting vm.BaseDir (baseDir): %q", baseDir)
	}

	vm.VDFName = strings.TrimPrefix(vm.VDFName, `.\`)

	vdfsbuilder.SanitizeVM(vm)

	if outFile != "" {
		vm.VDFName = outFile
		githubactions.Infof("Overwriting vm.VDFName (out): %q", outFile)
	}

	if tsOverrideStr != "" {
		location := time.Local
		if true /* *tsIsUtc */ {
			location = time.UTC
		}
		parsed, err := time.ParseInLocation("2006-01-02 15:04:05", tsOverrideStr, location)
		if err != nil {
			githubactions.Warningf("Failed to parse %q flag. %v", tsOverrideStr, err)
		} else {
			githubactions.Infof("Override: Timestamp set to %q (%s)\n", parsed.Format("2006-01-02 15:04:05"), location)
			vm.Timestamp = parsed
		}
	}

	if err := vm.Execute(); err != nil {
		githubactions.Fatalf("failed to execute %q. %v", inFile, err)
	}
}
