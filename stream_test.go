// Copyright (c) 2019 Faye Amacker. All rights reserved.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package cbor_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/fxamacker/cbor"
)

func TestDecoder(t *testing.T) {
	var buf bytes.Buffer
	for i := 0; i < 5; i++ {
		for _, tc := range unmarshalTests {
			buf.Write(tc.cborData)
		}
	}
	decoder := cbor.NewDecoder(&buf)
	bytesRead := 0
	for i := 0; i < 5; i++ {
		for _, tc := range unmarshalTests {
			var v interface{}
			if err := decoder.Decode(&v); err != nil {
				t.Fatalf("Decode() returns error %v", err)
			}
			if !reflect.DeepEqual(v, tc.emptyInterfaceValue) {
				t.Errorf("Decode() = %v (%T), want %v (%T)", v, v, tc.emptyInterfaceValue, tc.emptyInterfaceValue)
			}
			bytesRead += len(tc.cborData)
			if decoder.NumBytesRead() != bytesRead {
				t.Errorf("NumBytesRead() = %v, want %v", decoder.NumBytesRead(), bytesRead)
			}
		}
	}
	// no more data
	var v interface{}
	err := decoder.Decode(&v)
	if v != nil {
		t.Errorf("Decode() = %v (%T), want nil (no more data)", v, v)
	}
	if err != io.EOF {
		t.Errorf("Decode() returns error %v, want io.EOF (no more data)", err)
	}
}

func TestDecoderUnmarshalTypeError(t *testing.T) {
	var buf bytes.Buffer
	for i := 0; i < 5; i++ {
		for _, tc := range unmarshalTests {
			for j := 0; j < len(tc.wrongTypes)*2; j++ {
				buf.Write(tc.cborData)
			}
		}
	}
	decoder := cbor.NewDecoder(&buf)
	bytesRead := 0
	for i := 0; i < 5; i++ {
		for _, tc := range unmarshalTests {
			for _, typ := range tc.wrongTypes {
				v := reflect.New(typ)
				if err := decoder.Decode(v.Interface()); err == nil {
					t.Errorf("Decode(0x%0x) returns %v (%T), want UnmarshalTypeError", tc.cborData, v.Elem().Interface(), v.Elem().Interface())
				} else if _, ok := err.(*cbor.UnmarshalTypeError); !ok {
					t.Errorf("Decode(0x%0x) returns wrong error %s, want UnmarshalTypeError", tc.cborData, err.Error())
				}
				bytesRead += len(tc.cborData)
				if decoder.NumBytesRead() != bytesRead {
					t.Errorf("NumBytesRead() = %v, want %v", decoder.NumBytesRead(), bytesRead)
				}

				var vi interface{}
				if err := decoder.Decode(&vi); err != nil {
					t.Errorf("Decode() returns error %v", err)
				}
				if !reflect.DeepEqual(vi, tc.emptyInterfaceValue) {
					t.Errorf("Decode() = %v (%T), want %v (%T)", vi, vi, tc.emptyInterfaceValue, tc.emptyInterfaceValue)
				}
				bytesRead += len(tc.cborData)
				if decoder.NumBytesRead() != bytesRead {
					t.Errorf("NumBytesRead() = %v, want %v", decoder.NumBytesRead(), bytesRead)
				}
			}
		}
	}
	// no more data
	var v interface{}
	err := decoder.Decode(&v)
	if v != nil {
		t.Errorf("Decode() = %v (%T), want nil (no more data)", v, v)
	}
	if err != io.EOF {
		t.Errorf("Decode() returns error %v, want io.EOF (no more data)", err)
	}
}

func TestDecoderStructTag(t *testing.T) {
	type strc struct {
		A string `json:"x" cbor:"a"`
		B string `json:"y" cbor:"b"`
		C string `json:"z"`
	}
	want := strc{
		A: "A",
		B: "B",
		C: "C",
	}
	cborData := hexDecode("a36161614161626142617a6143") // {"a":"A", "b":"B", "z":"C"}

	var v strc
	dec := cbor.NewDecoder(bytes.NewReader(cborData))
	if err := dec.Decode(&v); err != nil {
		t.Errorf("Decode() returns error %v", err)
	}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("Decode() = %+v (%T), want %+v (%T)", v, v, want, want)
	}
}

func TestEncoder(t *testing.T) {
	var want bytes.Buffer
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{Canonical: true})
	for _, tc := range marshalTests {
		for _, value := range tc.values {
			want.Write(tc.cborData)

			if err := encoder.Encode(value); err != nil {
				t.Fatalf("Encode() returns error %v", err)
			}
		}
	}
	if !bytes.Equal(w.Bytes(), want.Bytes()) {
		t.Errorf("Encoding mismatch: got %v, want %v", w.Bytes(), want.Bytes())
	}
}

func TestEncoderError(t *testing.T) {
	testcases := []struct {
		name         string
		value        interface{}
		wantErrorMsg string
	}{
		{"channel can't be marshalled", make(chan bool), "cbor: unsupported type: chan bool"},
		{"function can't be marshalled", func(i int) int { return i * i }, "cbor: unsupported type: func(int) int"},
		{"complex can't be marshalled", complex(100, 8), "cbor: unsupported type: complex128"},
	}
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{})
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := encoder.Encode(&tc.value)
			if err == nil {
				t.Errorf("Encode(%v) doesn't return an error, want error %q", tc.value, tc.wantErrorMsg)
			} else if _, ok := err.(*cbor.UnsupportedTypeError); !ok {
				t.Errorf("Encode(%v) error type %T, want *cbor.UnsupportedTypeError", tc.value, err)
			} else if err.Error() != tc.wantErrorMsg {
				t.Errorf("Encode(%v) error %s, want %s", tc.value, err, tc.wantErrorMsg)
			}
		})
	}
	if w.Len() != 0 {
		t.Errorf("Encoder's writer has %d bytes of data, want empty data", w.Len())
	}
}

func TestIndefiniteByteString(t *testing.T) {
	want := hexDecode("5f42010243030405ff")
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{})
	if err := encoder.StartIndefiniteByteString(); err != nil {
		t.Fatalf("StartIndefiniteByteString() returns error %v", err)
	}
	if err := encoder.Encode([]byte{1, 2}); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.Encode([3]byte{3, 4, 5}); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.EndIndefinite(); err != nil {
		t.Fatalf("EndIndefinite() returns error %v", err)
	}
	if !bytes.Equal(w.Bytes(), want) {
		t.Errorf("Encoding mismatch: got %v, want %v", w.Bytes(), want)
	}
}

func TestIndefiniteByteStringError(t *testing.T) {
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{})
	if err := encoder.StartIndefiniteByteString(); err != nil {
		t.Fatalf("StartIndefiniteByteString() returns error %v", err)
	}
	if err := encoder.Encode([]int{1, 2}); err == nil {
		t.Errorf("Encode() expects error, got nil")
	} else if err.Error() != "cbor: cannot encode item type slice for indefinite-length byte string" {
		t.Errorf("Encode() error %v, want %s", err, "cbor: cannot encode item type slice for indefinite-length byte string")
	}
	if err := encoder.Encode("hello"); err == nil {
		t.Errorf("Encode() expects error, got nil")
	} else if err.Error() != "cbor: cannot encode item type string for indefinite-length byte string" {
		t.Errorf("Encode() error %v, want %s", err, "cbor: cannot encode item type string for indefinite-length byte string")
	}
}

func TestIndefiniteTextString(t *testing.T) {
	want := hexDecode("7f657374726561646d696e67ff")
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{})
	if err := encoder.StartIndefiniteTextString(); err != nil {
		t.Fatalf("StartIndefiniteTextString() returns error %v", err)
	}
	if err := encoder.Encode("strea"); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.Encode("ming"); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.EndIndefinite(); err != nil {
		t.Fatalf("EndIndefinite() returns error %v", err)
	}
	if !bytes.Equal(w.Bytes(), want) {
		t.Errorf("Encoding mismatch: got %v, want %v", w.Bytes(), want)
	}
}

func TestIndefiniteTextStringError(t *testing.T) {
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{})
	if err := encoder.StartIndefiniteTextString(); err != nil {
		t.Fatalf("StartIndefiniteTextString() returns error %v", err)
	}
	if err := encoder.Encode([]byte{1, 2}); err == nil {
		t.Errorf("Encode() expects error, got nil")
	} else if err.Error() != "cbor: cannot encode item type slice for indefinite-length text string" {
		t.Errorf("Encode() error %v, want %s", err, "cbor: cannot encode item type slice for indefinite-length text string")
	}
}

func TestIndefiniteArray(t *testing.T) {
	want := hexDecode("9f018202039f0405ffff")
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{})
	if err := encoder.StartIndefiniteArray(); err != nil {
		t.Fatalf("StartIndefiniteArray() returns error %v", err)
	}
	if err := encoder.Encode(1); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.Encode([]int{2, 3}); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.StartIndefiniteArray(); err != nil {
		t.Fatalf("StartIndefiniteArray() returns error %v", err)
	}
	if err := encoder.Encode(4); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.Encode(5); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.EndIndefinite(); err != nil {
		t.Fatalf("EndIndefinite() returns error %v", err)
	}
	if err := encoder.EndIndefinite(); err != nil {
		t.Fatalf("EndIndefinite() returns error %v", err)
	}
	if !bytes.Equal(w.Bytes(), want) {
		t.Errorf("Encoding mismatch: got %v, want %v", w.Bytes(), want)
	}
}

func TestIndefiniteMap(t *testing.T) {
	want := hexDecode("bf61610161629f0203ffff")
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{Canonical: true})
	if err := encoder.StartIndefiniteMap(); err != nil {
		t.Fatalf("StartIndefiniteMap() returns error %v", err)
	}
	if err := encoder.Encode("a"); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.Encode(1); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.Encode("b"); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.StartIndefiniteArray(); err != nil {
		t.Fatalf("StartIndefiniteArray() returns error %v", err)
	}
	if err := encoder.Encode(2); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.Encode(3); err != nil {
		t.Fatalf("Encode() returns error %v", err)
	}
	if err := encoder.EndIndefinite(); err != nil {
		t.Fatalf("EndIndefinite() returns error %v", err)
	}
	if err := encoder.EndIndefinite(); err != nil {
		t.Fatalf("EndIndefinite() returns error %v", err)
	}
	if !bytes.Equal(w.Bytes(), want) {
		t.Errorf("Encoding mismatch: got %v, want %v", w.Bytes(), want)
	}
}

func TestIndefiniteLengthError(t *testing.T) {
	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{})
	if err := encoder.StartIndefiniteByteString(); err != nil {
		t.Fatalf("StartIndefiniteByteString() returns error %v", err)
	}
	if err := encoder.EndIndefinite(); err != nil {
		t.Fatalf("EndIndefinite() returns error %v", err)
	}
	if err := encoder.EndIndefinite(); err == nil {
		t.Fatalf("EndIndefinite() doesn't return error")
	}
}

func TestEncoderStructTag(t *testing.T) {
	type strc struct {
		A string `json:"x" cbor:"a"`
		B string `json:"y" cbor:"b"`
		C string `json:"z"`
	}
	v := strc{
		A: "A",
		B: "B",
		C: "C",
	}
	want := hexDecode("a36161614161626142617a6143") // {"a":"A", "b":"B", "z":"C"}

	var w bytes.Buffer
	encoder := cbor.NewEncoder(&w, cbor.EncOptions{Canonical: true})
	if err := encoder.Encode(v); err != nil {
		t.Errorf("Encode(%+v) returns error %v", v, err)
	}
	if !bytes.Equal(w.Bytes(), want) {
		t.Errorf("Encoding mismatch: got %v, want %v", w.Bytes(), want)
	}
}
