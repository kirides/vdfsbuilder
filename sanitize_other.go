//go:build !windows

package vdfsbuilder

import (
	"github.com/kirides/vdfsbuilder/vdf"
	"os"
	"path/filepath"
)

func SanitizeVM(vm *vdf.VM) {
	var err error
	if vm.BaseDir == `.\` {
		vm.BaseDir, err = os.Getwd()
		if err != nil {
			panic(err)
		}
	}
	vm.VDFName = filepath.ToSlash(vm.VDFName)
	vm.BaseDir = filepath.ToSlash(vm.BaseDir)
}
