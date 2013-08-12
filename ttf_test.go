package ttf

import (
	"bytes"
	"testing"
)

var testFile = "Roboto_Regular.ttf"

func TestWrongFile(t *testing.T) {
	file := bytes.NewReader([]byte("just some text file"))
	_, err := Read(file)
	if err == nil {
		t.Error("Acctepted a non-TTF file")
	}

}

func TestReadFile(t *testing.T) {
	file := bytes.NewReader(roboto)
	_, err := Read(file)
	if err != nil {
		t.Error(err)
	}

}

func TestTables(t *testing.T) {
	file := bytes.NewReader([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x20})
	ttf, err := Read(file)
	if err != nil {
		t.Error(err)
	}

	tablesNum := ttf.TablesNum()
	if tablesNum != 0x20 {
		t.Errorf("Expected %v tables but got %v", 0x20, tablesNum)
	}
}
