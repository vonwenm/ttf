package ttf

import (
	"bytes"
	"encoding/binary"
	"io"
)

type ttfError string

func (e ttfError) Error() string {
	return "TTF Error: " + string(e)
}

type tableType uint32

const (
	CMAP tableType = iota
)

func (t tableType) String() string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(t))
	return "tableType(" + buf.String() + ")"
}

type tableData struct {
	Tag, Checksum, Offset, N uint32
}

type readerAt interface {
	io.Reader
	io.ReaderAt
}

type TTF struct {
	file   readerAt
	tables map[tableType]tableData
}

const (
	headerSize     = 4 + 4*2
	tableEntrySize = 4 * 4
)

func (ttf *TTF) readTableEntry(i int) error {
	var data tableData

	offset := headerSize + int64(i)*tableEntrySize
	tableEntry := io.NewSectionReader(ttf.file, offset, tableEntrySize)

	err := binary.Read(tableEntry, binary.BigEndian, &data)
	if err != nil {
		return err
	}

	ttf.tables[tableType(data.Tag)] = data

	return nil
}

type header struct {
	Filetype  uint32
	TablesNum uint16
	// 3 uint16 ignored
}

func (ttf *TTF) readTableDir() (err error) {
	var data header
	header := io.NewSectionReader(ttf.file, 0, headerSize)

	err = binary.Read(header, binary.BigEndian, &data)
	if err != nil {
		return
	}

	if data.Filetype != 0x74727565 && data.Filetype != 0x00010000 {
		return ttfError("Invalid magic number")
	}

	for i := 0; i < int(data.TablesNum); i++ {
		ttf.readTableEntry(i)
	}

	return
}

func Read(file readerAt) (TTF, error) {
	ttf := TTF{file, make(map[tableType]tableData)}
	err := ttf.readTableDir()

	return ttf, err
}

func (ttf TTF) TablesNum() int {
	return len(ttf.tables)
}
