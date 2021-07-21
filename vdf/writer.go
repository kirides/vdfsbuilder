package vdf

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unsafe"
)

type VM struct {
	Comment string
	BaseDir string
	VDFName string

	Files   []string
	Exclude []string
	Include []string

	masks []*regexp.Regexp
}

type FileEntry struct {
	Name, RelPath string
	Flags         EntryFlag
	Attr          EntryAttrib
	Size          int64
}

type Dirs struct {
	Name  string
	Attr  EntryAttrib
	Files []*FileEntry
	Dirs  []*Dirs
}

func (d *Dirs) AddDir(e *Dirs) {
	d.Dirs = append(d.Dirs, e)
}
func (d *Dirs) AddFile(e *FileEntry) {
	d.Files = append(d.Files, e)
}

func (d *Dirs) NumEntries() (int64, int) {
	fullSize := int64(0)
	entries := 0

	for _, v := range d.Dirs {
		s, e := v.NumEntries()
		fullSize += s
		entries += e
	}
	entries += len(d.Dirs)
	entries += len(d.Files)
	for _, v := range d.Files {
		fullSize += v.Size
	}
	return fullSize, entries
}

func getFileAttr(path string) EntryAttrib {
	return 0

	// p, err := syscall.UTF16PtrFromString(path)
	// if err != nil {
	// 	panic(err)
	// }

	// attr, err := syscall.GetFileAttributes(p)
	// if err != nil {
	// 	panic(err)
	// }
	// return EntryAttrib(attr) & EntryAttribMask
}

func (vm *VM) searchFiles(root, path string, list *Dirs) int {
	fileCount := 0
	fullPath := root
	result := 0
	if path != "" {
		fullPath = filepath.Join(root, path)
	}
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		info, err := entry.Info()
		if err != nil {
			panic(err)
		}
		subPath := filepath.Join(path, entry.Name())
		attr := getFileAttr(filepath.Join(root, subPath))
		// EntryAttrib(info.Mode() & fs.FileMode(EntryAttribMask))
		if entry.IsDir() {
			// subPath := filepath.Join(path, entry.Name())
			de := &Dirs{
				Name: name,
				Attr: attr,
			}
			if n := vm.searchFiles(root, subPath, de); n != 0 {
				list.AddDir(de)
				result += n
			}
		} else {
			found := false
			for _, f := range list.Files {
				if strings.EqualFold(f.Name, name) {
					found = true
					break
				}
			}
			if found {
				// Only add each name file once (??)
				break
			}
			ok := false
			for i := 0; i < len(vm.masks); i++ {
				if vm.masks[i].MatchString(subPath) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
			fe := &FileEntry{
				Name: name,
				Size: info.Size(),
				Attr: attr,
			}
			list.AddFile(fe)
			fileCount++
			result++
		}
	}
	return result
}

func entryName(n string) EntryName {
	var e EntryName
	n = strings.ToUpper(n)
	for i := len(n); i < len(e); i++ {
		e[i] = 0x20
	}
	copy(e[:], n)
	return e
}

type ExtendedEntryMetadata struct {
	EntryMetadata

	Path string
}

type VdfsTable []ExtendedEntryMetadata

func (vm *VM) readFilesFromList(list *Dirs, table VdfsTable, root, path string, index uint, dataPos *size_t) bool {
	result := true
	idx := index
	index += uint(len(list.Dirs) + len(list.Files))

	for i, v := range list.Dirs {
		e := ExtendedEntryMetadata{
			Path: "",
			EntryMetadata: EntryMetadata{
				Name:    entryName(v.Name),
				Offset:  size_t(index),
				Size:    0,
				Flags:   EntryFlagDirectory,
				Attribs: v.Attr,
			}}
		if len(list.Files) == 0 && i == len(list.Dirs)-1 {
			e.Flags |= EntryFlagLastEntry
		}
		table[idx] = e
		subPath := filepath.Join(path, v.Name)
		if !vm.readFilesFromList(v, table, root, subPath, index, dataPos) {
			result = false
			break
		}
		idx++
	}

	for i, v := range list.Files {
		e := ExtendedEntryMetadata{
			Path: filepath.Join(path, v.Name),
			EntryMetadata: EntryMetadata{
				Name:    entryName(v.Name),
				Offset:  size_t(*dataPos),
				Size:    size_t(v.Size),
				Flags:   0,
				Attribs: v.Attr,
			}}
		if i == len(list.Files)-1 {
			e.EntryMetadata.Flags |= EntryFlagLastEntry
		}
		table[idx] = e
		*dataPos += e.Size
		idx++
	}

	return result
}

func vdfDateTime(t time.Time) time_t {
	if unsafe.Sizeof(time_t(0)) == 8 {
		return time_t(t.Unix())
	}

	// calculate Fat DateTime

	/* https://stackoverflow.com/a/15763512

				   24                16                 8                 0
	+-+-+-+-+-+-+-+-+ +-+-+-+-+-+-+-+-+ +-+-+-+-+-+-+-+-+ +-+-+-+-+-+-+-+-+
	|Y|Y|Y|Y|Y|Y|Y|M| |M|M|M|D|D|D|D|D| |h|h|h|h|h|m|m|m| |m|m|m|s|s|s|s|s|
	+-+-+-+-+-+-+-+-+ +-+-+-+-+-+-+-+-+ +-+-+-+-+-+-+-+-+ +-+-+-+-+-+-+-+-+
	 \___________/ \_______/ \_______/   \_______/ \___________/ \_______/
		year        month       day         hour     minute        second

	The year is stored as an offset from 1980.
	Seconds are stored in two-second increments.
	(So if the "second" value is 15, it actually represents 30 seconds.)
	*/

	fdt := uint32(t.Year()-1980) << 25
	fdt |= uint32(t.Month()) << 21
	fdt |= uint32(t.Day()) << 16
	fdt |= uint32(t.Hour()) << 11
	fdt |= uint32(t.Minute()) << 5
	fdt |= uint32(t.Second()) >> 1
	return time_t(fdt)
}

func (vm *VM) Execute() {
	basePath := vm.BaseDir
	vm.masks = buildMasks(vm.Files)

	f, err := os.Create(vm.VDFName)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	comment := Comment{}
	for i := 0; i < len(comment); i++ {
		comment[i] = 0x1A
	}
	copy(comment[:], []byte(vm.Comment))

	version := Version{'P', 'S', 'V', 'D', 'S', 'C', '_', 'V', '2', '.', '0', '0', '\n', '\r', '\n', '\r'}
	// version3 := Version{'P', 'S', 'V', 'D', 'S', 'C', '_', 'V', '3', '.', '0', '0', '\n', '\r', '\n', '\r'}

	dirs := &Dirs{}
	nFiles := vm.searchFiles(basePath, "", dirs)
	dataSize, entryCount := dirs.NumEntries()

	nowFileTime := vdfDateTime(time.Now())
	header := Header{
		Comment: comment,
		Version: version,
		Params: Params{
			EntryCount:  uint32(entryCount),
			FileCount:   uint32(nFiles),
			TimeStamp:   nowFileTime,
			DataSize:    size_t(dataSize),
			TableOffset: uint32(unsafe.Sizeof(Header{})),
			EntrySize:   uint32(unsafe.Sizeof(EntryMetadata{})),
		}}
	binary.Write(f, binary.LittleEndian, header)

	tbl := make(VdfsTable, header.Params.EntryCount)
	tableSize := header.Params.EntryCount * header.Params.EntrySize
	dataPos := size_t(header.Params.TableOffset + tableSize)
	vm.readFilesFromList(dirs, tbl, basePath, "", 0, &dataPos)

	for _, v := range tbl {
		binary.Write(f, binary.LittleEndian, v.EntryMetadata)
	}
	curPos, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}
	f.Truncate(curPos + dataSize)

	for _, v := range tbl {
		if v.Flags&EntryFlagDirectory != 0 {
			continue
		}
		nf, err := os.Open(filepath.Join(basePath, v.Path))
		if err != nil {
			panic(err)
		}
		defer nf.Close()
		_, err = f.Seek(int64(v.Offset), io.SeekStart)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(f, nf)
		if err != nil {
			panic(err)
		}
		nf.Close()
	}

}

func buildMasks(files []string) []*regexp.Regexp {
	var result []*regexp.Regexp
	for _, f := range files {
		// TODO: support non recursive matching
		f = strings.TrimSuffix(f, " -r")
		expr := regexp.QuoteMeta(f)
		// if strings.HasSuffix(f, " -r") {
		// recurse
		expr = strings.ReplaceAll(expr, `\*`, `.*`)
		expr = strings.ReplaceAll(expr, `\?`, `.`)
		// }
		expr = "(?i)^" + expr + "$"
		result = append(result, regexp.MustCompile(expr))
	}
	return result
}

// func buildExcludes(excl []string) []*regexp.Regexp {
// 	var result []*regexp.Regexp

// 	for _, v := range excl {
// 		_ = v
// 	}

// 	return result
// }
