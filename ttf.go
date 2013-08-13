package ttf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type ttfError string

func (e ttfError) Error() string {
	return "TTF Error: " + string(e)
}

type tableType uint32

var (
	INVALID = tableType(0)
	HEAD    = tableTypeString("head")
	CMAP    = tableTypeString("cmap")
	GLYF    = tableTypeString("glyf")
	HHEA    = tableTypeString("hhea")
	HMTX    = tableTypeString("hmtx")
	LOCA    = tableTypeString("loca")
	MAXP    = tableTypeString("maxp")
	NAME    = tableTypeString("name")
	POST    = tableTypeString("post")
)

func tableTypeString(s string) (ret tableType) {
	if len(s) != 4 {
		return
	}

	buf := bytes.NewBufferString(s)
	binary.Read(buf, binary.BigEndian, &ret)
	return
}

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

func checksum(r io.Reader) (sum uint32) {
	for {
		var i uint32

		err := binary.Read(r, binary.BigEndian, &i)
		if err != nil {
			return
		}

		sum += i
	}

	panic("Reached unreachable path!?")
	return
}

var requiredTables = []tableType{CMAP, GLYF, HEAD, HHEA, HMTX, LOCA, MAXP, NAME, POST}

func (ttf TTF) fontChecksum() (checksum uint32, err error) {
	table, ok := ttf.tables[HEAD]
	if !ok {
		return 0, ttfError("table 'head' not found")
	}

	sect := io.NewSectionReader(ttf.file, int64(table.Offset+8), 4)
	err = binary.Read(sect, binary.BigEndian, &checksum)
	return
}

func (ttf TTF) checkTableChecksum(ttype tableType, table tableData) error {
	N := 4 * ((3 + table.N) / 4)
	sect := io.NewSectionReader(ttf.file, int64(table.Offset), int64(N))
	sum := checksum(sect)

	if ttype == HEAD {
		fontSum, err := ttf.fontChecksum()
		if err != nil {
			return err
		}
		sum -= fontSum
	}

	if table.Checksum != sum {
		return ttfError(fmt.Sprintf("Table '%v' checksum failed.", ttype))
	}

	return nil
}

func (ttf TTF) checkFontChecksum() error {
	font := io.NewSectionReader(ttf.file, 0, math.MaxInt64)
	fsum, err := ttf.fontChecksum()
	if err != nil {
		return err
	}

	sum := 0xB1B0AFBA - checksum(font) + fsum
	if fsum != sum {
		return ttfError("Font checksum failed")
	}

	return nil
}

func (ttf TTF) Check() error {

	for _, reqType := range requiredTables {
		if _, ok := ttf.tables[reqType]; !ok {
			return ttfError("Missing required table " + reqType.String())
		}
	}

	for ttype, table := range ttf.tables {
		if err := ttf.checkTableChecksum(ttype, table); err != nil {
			return err
		}
	}

	err := ttf.checkFontChecksum()

	return err
}
