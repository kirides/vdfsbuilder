package vdf

type EntryName [0x3F + 1]byte

// func (c EntryName) String() string {
// 	if i := bytes.IndexByte(c[:], 0x00); i != -1 {
// 		return "ROOT"
// 	}
// 	return strings.TrimRight(string([]byte(c[:0x3F])), " ")
// }
// func (c EntryName) MarshalText() ([]byte, error) {
// 	return []byte(c.String()), nil
// }

type EntryMetadata struct {
	Name    EntryName
	Offset  uint32
	Size    uint32
	Flags   EntryFlag
	Attribs EntryAttrib
}

type Comment [0xFF + 1]byte

// func (c Comment) String() string {
// 	if i := bytes.IndexByte(c[:], 0x1A); i != -1 {
// 		return string([]byte(c[:i]))
// 	}
// 	return string([]byte(c[:0xFF]))
// }

// func (c Comment) MarshalText() ([]byte, error) {
// 	return []byte(c.String()), nil
// }

type Version [0x0F + 1]byte

// func (c Version) String() string {
// 	if i := bytes.IndexByte(c[:], 0x0A); i != -1 {
// 		return string([]byte(c[:i]))
// 	}
// 	return string([]byte(c[:0x0F]))
// }

// func (c Version) MarshalText() (text []byte, err error) {
// 	return []byte(c.String()), nil
// }

type Params struct {
	EntryCount  uint32
	FileCount   uint32
	TimeStamp   uint32
	DataSize    uint32
	TableOffset uint32
	EntrySize   uint32
}
type Header struct {
	Comment Comment
	Version Version
	Params  Params
}

type EntryFlag uint32

const (
	EntryFlagDirectory EntryFlag = 0x80000000
	EntryFlagLastEntry EntryFlag = 0x40000000
)

type EntryAttrib uint32

const (
	EntryAttribReadOnly EntryAttrib = 1
	EntryAttribHidden   EntryAttrib = 2
	EntryAttribSystem   EntryAttrib = 4
	EntryAttribArchive  EntryAttrib = 32

	EntryAttribMask EntryAttrib = EntryAttribReadOnly |
		EntryAttribHidden |
		EntryAttribSystem |
		EntryAttribArchive
)
