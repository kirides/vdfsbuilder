package vdf

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unsafe"
)

type VM struct {
	Comment   string
	BaseDir   string
	VDFName   string
	Timestamp time.Time

	Files   []string
	Exclude []string
	Include []string

	fileMasks            []*regexp.Regexp
	excludeMasks         []*regexp.Regexp
	includeMasks         []*regexp.Regexp
	fileHashToDataOffset map[string]int64
}

type fileEntry struct {
	Name, RelPath string
	Flags         EntryFlag
	Attr          EntryAttrib
	Size          int64
}
type dirEntry struct {
	Name  string
	Attr  EntryAttrib
	Files []*fileEntry
	Dirs  []*dirEntry
}

func (d *dirEntry) addDir(e *dirEntry)   { d.Dirs = append(d.Dirs, e) }
func (d *dirEntry) addFile(e *fileEntry) { d.Files = append(d.Files, e) }

func (d *dirEntry) numEntries() (int64, int) {
	fullSize := int64(0)
	entries := 0

	for _, v := range d.Dirs {
		s, e := v.numEntries()
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

func getFileAttr(entry fs.DirEntry) EntryAttrib {
	if entry.IsDir() {
		return 0 // same as GothicVDFS
	}
	const FILEATTRIB_ARCHIVE = 0x20
	return FILEATTRIB_ARCHIVE

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

func (vm *VM) matchesMasks(fullPath string) bool {
	fullPath = filepath.ToSlash(fullPath)
	// First try to include any file that matches [FILES]
	shouldInclude := slices.ContainsFunc(vm.fileMasks, func(rx *regexp.Regexp) bool {
		return rx.MatchString(fullPath)
	})

	wasExcluded := false
	if shouldInclude {
		// then figure out if it should be [EXCLUDE]d
		if slices.ContainsFunc(vm.excludeMasks, func(rx *regexp.Regexp) bool {
			return rx.MatchString(fullPath)
		}) {
			shouldInclude = false
			wasExcluded = true
		}
	}

	if wasExcluded {
		// And if it WAS excluded, check if we still should [INCLUDE] it
		shouldInclude = slices.ContainsFunc(vm.includeMasks, func(rx *regexp.Regexp) bool {
			return rx.MatchString(fullPath)
		})
	}

	return shouldInclude
}

func (vm *VM) searchFiles(root, path string, list *dirEntry) int {
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

		attr := getFileAttr(entry)
		if entry.IsDir() {
			// subPath := filepath.Join(path, entry.Name())
			de := &dirEntry{
				Name: name,
				Attr: attr,
			}
			if n := vm.searchFiles(root, subPath, de); n != 0 {
				list.addDir(de)
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
			if !vm.matchesMasks(subPath) {
				continue
			}
			fe := &fileEntry{
				Name: name,
				Size: info.Size(),
				Attr: attr,
			}
			list.addFile(fe)
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

type vdfsTable []ExtendedEntryMetadata

func (vm *VM) readFilesFromList(list *dirEntry, f *os.File, table vdfsTable, root, path string, index *uint, dataPos *size_t) bool {
	result := true
	idx := *index
	*index += uint(len(list.Dirs) + len(list.Files))

	for i, v := range list.Dirs {
		subPath := filepath.Join(path, v.Name)
		e := ExtendedEntryMetadata{
			Path: "",
			EntryMetadata: EntryMetadata{
				Name:    entryName(v.Name),
				Offset:  size_t(*index),
				Size:    0,
				Flags:   EntryFlagDirectory,
				Attribs: v.Attr,
			}}
		if len(list.Files) == 0 && i == len(list.Dirs)-1 {
			e.Flags |= EntryFlagLastEntry
		}
		table[idx] = e
		if !vm.readFilesFromList(v, f, table, root, subPath, index, dataPos) {
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

		if pos, ok := vm.tryGetExistingPos(root, path, v.Name); ok {
			e.Offset = size_t(pos)
			table[idx] = e
			idx++
			continue
		}

		table[idx] = e
		hash, ok := vm.appendDataFromDisk(f, root, path, v.Name)
		if !ok {
			fmt.Fprintf(os.Stdout, "could not process %q\n", v.Name)
			return false
		}
		vm.fileHashToDataOffset[hash] = int64(*dataPos)
		*dataPos += e.Size
		idx++
	}

	return result
}

func (vm *VM) tryGetExistingPos(root, path, name string) (int64, bool) {
	fullPath := filepath.Join(root, path, name)
	src, err := os.Open(fullPath)
	if err != nil {
		return 0, false
	}
	defer src.Close()

	hash, err := hashFile(src)
	if err != nil {
		return 0, false
	}
	if pos, ok := vm.fileHashToDataOffset[hash]; ok {
		return pos, true
	}
	return 0, false
}

func getHasher() hash.Hash {
	return sha256.New()
}

func hashFile(f *os.File) (string, error) {
	hasher := getHasher()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (vm *VM) appendDataFromDisk(f *os.File, root, path, name string) (string, bool) {
	fullPath := filepath.Join(root, path, name)
	src, err := os.Open(fullPath)
	if err != nil {
		return "", false
	}
	defer src.Close()

	hasher := getHasher()
	if _, err := io.Copy(io.MultiWriter(f, hasher), src); err != nil {
		return "", false
	}

	return hex.EncodeToString(hasher.Sum(nil)), true
}

func vdfDateTime(t time.Time) time_t {
	// TODO: once gothic overflows 2038
	// probably someone will mod in int64 timestamps
	// and require re-packing all VDFs

	// if unsafe.Sizeof(time_t(0)) == 8 {
	// 	return time_t(t.Unix())
	// }

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

func comment(c string) Comment {
	maxLen := int(unsafe.Sizeof(Comment{}))
	if len(c) > maxLen {
		c = c[:maxLen]
	}
	comment := Comment{}
	copy(comment[:], []byte(c))
	for i := len(c); i < len(comment); i++ {
		comment[i] = 0x1A
	}
	return comment
}

func (vm *VM) Execute() error {
	basePath := vm.BaseDir
	vm.fileMasks = buildMasks(vm.Files)
	vm.excludeMasks = buildMasks(vm.Exclude)
	vm.includeMasks = buildMasks(vm.Include)
	vm.fileHashToDataOffset = make(map[string]int64)

	f, err := os.Create(vm.VDFName)
	if err != nil {
		return fmt.Errorf("failed to create output. %w", err)
	}
	defer f.Close()

	version := Version{'P', 'S', 'V', 'D', 'S', 'C', '_', 'V', '2', '.', '0', '0', '\n', '\r', '\n', '\r'}
	// version3 := Version{'P', 'S', 'V', 'D', 'S', 'C', '_', 'V', '3', '.', '0', '0', '\n', '\r', '\n', '\r'}

	rootEntry := &dirEntry{}
	nFiles := vm.searchFiles(basePath, "", rootEntry)
	dataSize, entryCount := rootEntry.numEntries()

	nowFileTime := vdfDateTime(vm.Timestamp)
	header := Header{
		Comment: comment(vm.Comment),
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

	tbl := make(vdfsTable, header.Params.EntryCount)
	tableSize := header.Params.EntryCount * header.Params.EntrySize
	dataPos := size_t(header.Params.TableOffset + tableSize)

	if err := f.Truncate(int64(dataPos)); err != nil {
		return fmt.Errorf("could not truncate to fit data. %w", err)
	}

	if _, err := f.Seek(int64(dataPos), io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to data offset. %w", err)
	}

	startIndex := uint(0)
	vm.readFilesFromList(rootEntry, f, tbl, basePath, "", &startIndex, &dataPos)

	if _, err := f.Seek(int64(header.Params.TableOffset), io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to table offset. %w", err)
	}
	for _, v := range tbl {
		if err := binary.Write(f, binary.LittleEndian, v.EntryMetadata); err != nil {
			return fmt.Errorf("failed to write table entry. %q: %w", v.Name, err)
		}
	}
	return nil
}

func buildMasks(files []string) []*regexp.Regexp {
	var result []*regexp.Regexp
	for _, f := range files {
		// TODO: support non recursive matching
		recursive := strings.HasSuffix(f, " -r")
		f = strings.TrimSuffix(f, " -r")

		// clear any sole leading path delimitters
		f = filepath.ToSlash(f)
		f = strings.TrimLeft(f, "/")

		expr := regexp.QuoteMeta(f)
		// if strings.HasSuffix(f, " -r") {
		// recurse
		expr = strings.ReplaceAll(expr, `\*`, `.*`)
		expr = strings.ReplaceAll(expr, `\?`, `.`)
		// }
		if recursive {
			// match in any directory or at the relative root
			expr = `(?i)(?:^|\/)` + expr + "$"
		} else {
			// match against the full relative path
			expr = "(?i)^" + expr + "$"
		}
		result = append(result, regexp.MustCompile(expr))
	}
	return result
}
