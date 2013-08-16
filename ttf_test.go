package ttf

import (
	"bytes"
	"fmt"
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
	file := bytes.NewReader(roboto)
	ttf, err := Read(file)
	if err != nil {
		t.Error(err)
	}

	exp := 0x11
	tablesNum := ttf.TablesNum()
	if tablesNum != exp {
		t.Errorf("Expected %v tables but got %v", exp, tablesNum)
	}
}

func TestCheck(t *testing.T) {
	file := bytes.NewReader(roboto)
	ttf, err := Read(file)

	if err != nil {
		t.Error(err)
	}

	if err := ttf.Check(); err != nil {
		t.Error(err)
	}
}

func TestMapGlyph(t *testing.T) {
	file := bytes.NewReader(roboto)
	ttf, err := Read(file)

	if err != nil {
		t.Error(err)
	}

	glyph, err := ttf.MapGlyph('A')
	if err != nil {
		t.Error(err)
	}

	exp := 36
	if glyph != exp {
		t.Error(fmt.Sprint("For 'A' got glyph ", glyph, " expected ", exp))
	}
}
