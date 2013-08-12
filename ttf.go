package ttf

import (
	"encoding/binary"
	"io"
)

type ttfError string

func (e ttfError) Error() string {
	return "TTF Error: " + string(e)
}

type readerAt interface {
	io.Reader
	io.ReaderAt
}

type TTF struct {
	file      readerAt
	numTables uint16
}

func (ttf *TTF) readTableDir() (err error) {
	var scalarType uint32

	err = binary.Read(ttf.file, binary.BigEndian, &scalarType)
	if err != nil {
		return
	}
	if scalarType != 0x74727565 && scalarType != 0x00010000 {
		return ttfError("Invalid magic number")
	}

	err = binary.Read(ttf.file, binary.BigEndian, &ttf.numTables)
	if err != nil {
		return
	}

	return
}

func Read(file readerAt) (TTF, error) {
	ttf := TTF{file, 0}
	err := ttf.readTableDir()

	return ttf, err
}

func (ttf TTF) TablesNum() int {
	return int(ttf.numTables)
}
