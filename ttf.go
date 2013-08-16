package ttf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

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
	mapper mapper
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
		return errors.New("Invalid magic number")
	}

	for i := 0; i < int(data.TablesNum); i++ {
		ttf.readTableEntry(i)
	}

	return
}

func Read(file readerAt) (*TTF, error) {
	ttf := &TTF{file, make(map[tableType]tableData), nil}
	err := ttf.readTableDir()

	return ttf, err
}

func (ttf *TTF) TablesNum() int {
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

func (ttf *TTF) fontChecksum() (checksum uint32, err error) {
	table, ok := ttf.tables[HEAD]
	if !ok {
		return 0, errors.New("table 'head' not found")
	}

	sect := io.NewSectionReader(ttf.file, int64(table.Offset+8), 4)
	err = binary.Read(sect, binary.BigEndian, &checksum)
	return
}

func (ttf *TTF) tableReader(ttype tableType) (*io.SectionReader, error) {
	table, ok := ttf.tables[ttype]
	if !ok {
		return nil, fmt.Errorf("Table %v not found", ttype)
	}

	return io.NewSectionReader(ttf.file, int64(table.Offset), int64(table.N)), nil
}

func (ttf *TTF) checkTableChecksum(ttype tableType, table tableData) error {
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
		return fmt.Errorf("Table '%v' checksum failed.", ttype)
	}

	return nil
}

func (ttf *TTF) checkFontChecksum() error {
	font := io.NewSectionReader(ttf.file, 0, math.MaxInt64)
	fsum, err := ttf.fontChecksum()
	if err != nil {
		return err
	}

	sum := 0xB1B0AFBA - checksum(font) + fsum
	if fsum != sum {
		return errors.New("Font checksum failed")
	}

	return nil
}

func (ttf *TTF) Check() error {
	for _, reqType := range requiredTables {
		if _, ok := ttf.tables[reqType]; !ok {
			return fmt.Errorf("Missing required table %v", reqType)
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

type cmapIndex struct {
	Version, TablesNum uint16
}

type cmapSubtable struct {
	Platform platformType
	Specific uint16
	Offset   uint32
}

type cmapSubtableHeader struct {
	Type, Len uint16
}

type cmapSubtable4 struct {
	Type, Len, Lang, Count2, X0, X1, X2 uint16
}

type mapper interface {
	Map(r rune) (int, error)
}

type mapper4 struct {
	Ranges []range4
	Direct map[uint16]uint16
}

type range4 struct {
	Start, End, Delta uint16
}

func newMapper4(s *io.SectionReader) (map4 *mapper4, err error) {
	var submap cmapSubtable4
	err = binary.Read(s, binary.BigEndian, &submap)
	if err != nil {
		return
	}

	if submap.Type != 4 {
		panic("newMapper4 called with submap.Type != 4")
	}

	count := int(submap.Count2 / 2)
	end := make([]uint16, count)
	start := make([]uint16, count)
	delta := make([]uint16, count)
	offset := make([]uint16, count)

	err = binary.Read(s, binary.BigEndian, &end)
	if err != nil {
		return
	}

	var X uint16
	err = binary.Read(s, binary.BigEndian, &X)
	if err != nil {
		return
	}

	err = binary.Read(s, binary.BigEndian, &start)
	if err != nil {
		return
	}

	err = binary.Read(s, binary.BigEndian, &delta)
	if err != nil {
		return
	}

	err = binary.Read(s, binary.BigEndian, &offset)
	if err != nil {
		return
	}

	ranges := make([]range4, 0)
	direct := make(map[uint16]uint16)
	for i := 0; i < count; i++ {
		if offset[i] == 0 {
			ranges = append(ranges, range4{start[i], end[i], delta[i]})
		} else {
			//Error: range offset not implemented
			panic("Range offset mapping not implemented")
		}
	}

	return &mapper4{ranges, direct}, nil
}

func (m *mapper4) Map(r rune) (int, error) {
	if r > 0xFFFF {
		return 0, errors.New("Unicode rune out of range")
	}

	r16 := uint16(r)

	g, ok := m.Direct[r16]
	if ok {
		return int(g), nil
	}

	for _, rng := range m.Ranges {
		if rng.End < r16 {
			continue
		}
		if rng.Start > r16 {
			return 0, nil
		}
		return int(rng.Delta + r16), nil
	}

	return 0, nil
}

func (ttf *TTF) initMap(platform platformType) error {
	cmap, err := ttf.tableReader(CMAP)
	if err != nil {
		return err
	}

	var index cmapIndex
	err = binary.Read(cmap, binary.BigEndian, &index)
	if err != nil {
		return err
	}

	var subtable cmapSubtable
	for i := 0; i < int(index.TablesNum); i++ {
		err = binary.Read(cmap, binary.BigEndian, &subtable)
		if err != nil {
			return err
		}
		if subtable.Platform == platform {
			break
		}
	}

	submap := io.NewSectionReader(cmap, int64(subtable.Offset), 4)

	var header cmapSubtableHeader
	err = binary.Read(submap, binary.BigEndian, &header)
	if err != nil {
		return err
	}

	submap = io.NewSectionReader(cmap, int64(subtable.Offset), int64(header.Len))

	switch header.Type {
	case 4:
		ttf.mapper, err = newMapper4(submap)
	default:
		return fmt.Errorf("Unsupported cmap subtable type %v", header.Type)
	}

	return err

}

type platformType uint16

var (
	UNICODE platformType = 0
	MAC                  = 1
	WINDOWS              = 4
)

func (ttf *TTF) MapGlyph(r rune) (int, error) {
	if ttf.mapper == nil {
		ttf.initMap(UNICODE)
	}
	return ttf.mapper.Map(r)
}
