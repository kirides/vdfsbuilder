package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

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
	tsOverrideStr := flag.String("ts", "", "a Timestamp in the format \"YYYY-MM-dd HH:mm:ss\". E.g \"2021-11-28 12:31:40\"")
	// tsIsUtc := flag.Bool("utc", true, "if the \"ts\" argument should be interpreted as UTC time.")
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

	vmTimestamp := time.Now()
	if *tsOverrideStr != "" {
		location := time.Local
		if true /* *tsIsUtc */ {
			location = time.UTC
		}
		parsed, err := time.ParseInLocation("2006-01-02 15:04:05", *tsOverrideStr, location)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse %q flag. %v", *tsOverrideStr, err)
			os.Exit(2)
			return
		}
		fmt.Fprintf(os.Stdout, "Override: Timestamp set to %q (%s)\n", parsed.Format("2006-01-02 15:04:05"), location)
		vmTimestamp = parsed
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

	vm.Timestamp = vmTimestamp

	wd, _ := os.Getwd()
	fmt.Fprintf(os.Stdout, "working directory: %q\n", wd)

	if err := vm.Execute(); err != nil {
		log.Fatalf("failed to execute %q. %v", args[0], err)
	}
}
