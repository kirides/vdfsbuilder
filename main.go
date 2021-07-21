package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"vdfsbuilder/vdf"
)

func main() {
	vm, err := vdf.ParseVM(os.Args[1])
	if err != nil {
		panic(err)
	}
	// allow for custom base directory
	if len(os.Args) > 2 {
		vm.BaseDir = os.Args[2]
	}

	if runtime.GOOS != "windows" {
		vm.VDFName = filepath.ToSlash(vm.VDFName)
		if vm.BaseDir == `.\` {
			vm.BaseDir, err = os.Getwd()
			if err != nil {
				panic(err)
			}
		}
		vm.BaseDir = filepath.ToSlash(vm.BaseDir)
		vm.VDFName = filepath.ToSlash(strings.TrimPrefix(vm.VDFName, `.\`))
	}
	// fmt.Printf("%#v", vm)
	vm.Execute()
}
