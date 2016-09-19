package id

import (
	"testing"
)

func TestID(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatal(err)
	}

	nextID, err := New()
	if err != nil {
		t.Fatal(err)
	}

	if id == nextID {
		t.Error("equal consecutive IDs")
	}
}

func TestIDString(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatal(err)
	}

	idText, err := id.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if idString := id.String(); idString != string(idText) {
		t.Errorf("incorrect ID string value\n%s\n%s\n", idString, string(idText))
	}
}

func TestIDTextMarshall(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatal(err)
	}

	idText, err := id.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	textID := ID{}
	if err := textID.UnmarshalText(idText); err != nil {
		t.Fatal(err)
	}

	if id != textID {
		t.Errorf("incorrect unmarshalled ID value\n%v\n%v\n", id, textID)
	}
}

func TestIDBinaryMarshall(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatal(err)
	}

	idBinary, err := id.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	binaryID := ID{}
	if err := binaryID.UnmarshalText(idBinary); err != nil {
		t.Fatal(err)
	}

	if id != binaryID {
		t.Errorf("incorrect unmarshalled ID value\n%v\n%v\n", id, binaryID)
	}
}
