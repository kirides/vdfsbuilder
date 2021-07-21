package vdf

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	parseInitial = iota
	parseBegin
	parseEnd
	parseFiles
	parseExclude
	parseInclude
)

func switchState(state int, buffer []byte) (bool, int) {
	newState := state
	if bytes.Equal(buffer, []byte("[BEGINVDF]")) {
		newState = parseBegin
	} else if bytes.Equal(buffer, []byte("[FILES]")) {
		newState = parseFiles
	} else if bytes.Equal(buffer, []byte("[EXCLUDE]")) {
		newState = parseExclude
	} else if bytes.Equal(buffer, []byte("[INCLUDE]")) {
		newState = parseInclude
	} else if bytes.Equal(buffer, []byte("[ENDVDF]")) {
		newState = parseEnd
	}

	return state != newState, newState
}

func ParseVM(path string) (*VM, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	vm := &VM{}
	state := parseInitial
	for s.Scan() {
		if bytes.HasPrefix(s.Bytes(), []byte(";")) {
			continue
		}
		if len(bytes.TrimSpace(s.Bytes())) == 0 {
			continue
		}
		if yes, new := switchState(state, s.Bytes()); yes {
			state = new
			continue
		}
		switch state {
		case parseInitial, parseEnd:
			// Nothing to do
		case parseBegin:
			if bytes.HasPrefix(s.Bytes(), []byte("Comment=")) {
				vm.Comment = string(bytes.TrimLeft(s.Bytes(), "Comment="))
			} else if bytes.HasPrefix(s.Bytes(), []byte("BaseDir=")) {
				vm.BaseDir = string(bytes.TrimLeft(s.Bytes(), "BaseDir="))
			} else if bytes.HasPrefix(s.Bytes(), []byte("VDFName=")) {
				vm.VDFName = string(bytes.TrimLeft(s.Bytes(), "VDFName="))
			}
		case parseFiles:
			vm.Files = append(vm.Files, strings.ReplaceAll(s.Text(), `\`, string(filepath.Separator)))
		case parseExclude:
			vm.Exclude = append(vm.Exclude, strings.ReplaceAll(s.Text(), `\`, string(filepath.Separator)))
		case parseInclude:
			vm.Include = append(vm.Include, strings.ReplaceAll(s.Text(), `\`, string(filepath.Separator)))
		}
	}
	// should always be at the end
	if state == parseEnd {
		return vm, nil
	}
	return nil, errors.New("failed to parse the script")
}
