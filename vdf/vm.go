package vdf

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type parserState int

const (
	parseInitial parserState = iota
	parseBegin
	parseEnd
	parseFiles
	parseExclude
	parseInclude
)

type vdfSection struct {
	Identifier []byte
	state      parserState
}

var (
	sectionBeginVdf = vdfSection{[]byte("[BEGINVDF]"), parseBegin}
	sectionFiles    = vdfSection{[]byte("[FILES]"), parseFiles}
	sectionExclude  = vdfSection{[]byte("[EXCLUDE]"), parseExclude}
	sectionInclude  = vdfSection{[]byte("[INCLUDE]"), parseInclude}
	sectionEndVdf   = vdfSection{[]byte("[ENDVDF]"), parseEnd}

	sections = []vdfSection{
		sectionBeginVdf,
		sectionFiles,
		sectionExclude,
		sectionInclude,
		sectionEndVdf,
	}
)

func switchState(state parserState, buffer []byte) (bool, parserState) {
	newState := state
	for _, v := range sections {
		if len(buffer) >= len(v.Identifier) &&
			bytes.Equal(buffer[0:len(v.Identifier)], v.Identifier) {
			newState = v.state
		}
	}

	return state != newState, newState
}

func ParseVM(path string) (*VM, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseVM(f)
}

func parseVM(r io.Reader) (*VM, error) {
	s := bufio.NewScanner(r)
	vm := &VM{
		fileHashToDataOffset: make(map[string]int64),
	}
	state := parseInitial
	for s.Scan() {
		if bytes.HasPrefix(s.Bytes(), []byte(";")) {
			continue
		}
		trimmedLine := bytes.TrimSpace(s.Bytes())
		if len(trimmedLine) == 0 {
			continue
		}
		if yes, new := switchState(state, trimmedLine); yes {
			state = new
			continue
		}
		switch state {
		case parseInitial, parseEnd:
			// Nothing to do
		case parseBegin:
			if bytes.HasPrefix(s.Bytes(), []byte("Comment=")) {
				vm.Comment = string(bytes.TrimPrefix(s.Bytes(), []byte("Comment=")))
				vm.Comment = strings.ReplaceAll(vm.Comment, `%%N`, "\r\n")
			} else if bytes.HasPrefix(s.Bytes(), []byte("BaseDir=")) {
				vm.BaseDir = string(bytes.TrimPrefix(s.Bytes(), []byte("BaseDir=")))
			} else if bytes.HasPrefix(s.Bytes(), []byte("VDFName=")) {
				vm.VDFName = string(bytes.TrimPrefix(s.Bytes(), []byte("VDFName=")))
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
