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

	g := 'A'
	exp := 36
	glyph, err := ttf.MapGlyph(g)
	if err != nil {
		t.Error(err)
	}

	if glyph != exp {
		t.Errorf("For '%v' got glyph %v expected %v", g, glyph, exp)
	}
}
