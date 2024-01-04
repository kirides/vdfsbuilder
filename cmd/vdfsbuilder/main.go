package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/kirides/vdfsbuilder"
	"github.com/kirides/vdfsbuilder/vdf"
)

func invocation() string {
	if runtime.GOOS == "windows" {
		return "vdfsbuilder.exe"
	}
	return "./vdfsbuilder"
}

func main() {
	outFile := flag.String("o", "", "override output filepath")
	baseDir := flag.String("b", "", "base directory (substitution for \".\\\")")

	log.SetOutput(os.Stdout)

	flag.Usage = func() {
		fmt.Println("example:")
		fmt.Printf("%s [options] *.vm\n", invocation())
		fmt.Println()
		fmt.Println("options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		flag.PrintDefaults()
		os.Exit(1)
		return
	}

	vm, err := vdf.ParseVM(args[0])
	if err != nil {
		log.Fatalf("failed to parse input. %v", err)
	}
	// allow for custom base directory
	if *baseDir != "" {
		vm.BaseDir = *baseDir
	}

	vm.VDFName = strings.TrimPrefix(vm.VDFName, `.\`)

	vdfsbuilder.SanitizeVM(vm)

	if *outFile != "" {
		vm.VDFName = *outFile
	}

	wd, _ := os.Getwd()
	fmt.Fprintf(os.Stdout, "working directory: %q\n", wd)

	if err := vm.Execute(); err != nil {
		log.Fatalf("failed to execute %q. %v", args[0], err)
	}
}
