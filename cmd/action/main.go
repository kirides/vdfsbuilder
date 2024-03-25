package main

import (
	"strings"

	"github.com/kirides/vdfsbuilder"
	"github.com/kirides/vdfsbuilder/vdf"

	"github.com/sethvargo/go-githubactions"
)

func main() {
	inFile := githubactions.GetInput("in")
	outFile := githubactions.GetInput("out")
	baseDir := githubactions.GetInput("baseDir")

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
	if err := vm.Execute(); err != nil {
		githubactions.Fatalf("failed to execute %q. %v", inFile, err)
	}
}
