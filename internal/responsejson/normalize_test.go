package responsejson

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/http"
	"testing"
	"unicode/utf16"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func encoded(s string, width int, order binary.ByteOrder, bom bool) []byte {
	var b []byte
	if bom {
		if width == 2 {
			if order == binary.LittleEndian {
				b = []byte{255, 254}
			} else {
				b = []byte{254, 255}
			}
		} else if order == binary.LittleEndian {
			b = []byte{255, 254, 0, 0}
		} else {
			b = []byte{0, 0, 254, 255}
		}
	}
	if width == 2 {
		for _, r := range utf16.Encode([]rune(s)) {
			var x [2]byte
			order.PutUint16(x[:], r)
			b = append(b, x[:]...)
		}
	} else {
		for _, r := range s {
			var x [4]byte
			order.PutUint32(x[:], uint32(r))
			b = append(b, x[:]...)
		}
	}
	return b
}
func TestNormalizeSupportedCharsetsOwnBytesAndKeepHeaders(t *testing.T) {
	text := `{"output":"café🦫"}`
	for _, tc := range []struct {
		name  string
		width int
		order binary.ByteOrder
		bom   bool
	}{
		{"UTF-16", 2, binary.LittleEndian, true}, {"utf_16", 2, binary.BigEndian, true},
		{"utf-16-le", 2, binary.LittleEndian, false}, {"utf-16be", 2, binary.BigEndian, false},
		{"utf-32", 4, binary.LittleEndian, true}, {"UTF-32", 4, binary.BigEndian, true},
		{"utf-32le", 4, binary.LittleEndian, false}, {"utf-32-be", 4, binary.BigEndian, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := encoded(text, tc.width, tc.order, tc.bom)
			original := bytes.Clone(raw)
			header := http.Header{"Content-Type": []string{`application/json; charset="` + tc.name + `"`}}
			out, err := Normalize(raw, header)
			if err != nil || string(out) != text || !bytes.Equal(raw, original) {
				t.Fatal(string(out), err)
			}
			out[0] = '!'
			if !bytes.Equal(raw, original) || header.Get("Content-Type") != `application/json; charset="`+tc.name+`"` {
				t.Fatal("unowned result or mutated header")
			}
		})
	}
	for _, name := range []string{"ISO-8859-1", "latin1", "iso8859_1", "cp819"} {
		raw := []byte{'"', 0x80, 0xe9, '"'}
		out, err := Normalize(raw, http.Header{"Content-Type": []string{"application/json;charset=" + name}})
		if err != nil || string(out) != "\"\u0080é\"" {
			t.Fatal(name, string(out), err)
		}
	}
}
func TestNormalizeBoundariesPreserveUnclaimedDecoderDomain(t *testing.T) {
	raw := []byte("raw\xff")
	for _, header := range []http.Header{nil, {"Content-Type": []string{"application/json"}}, {"Content-Type": []string{"application/json;charset=utf-8"}}} {
		out, err := Normalize(raw, header)
		if err != nil || !bytes.Equal(out, raw) {
			t.Fatal(out, err)
		}
		out[0] = '!'
		if raw[0] != 'r' {
			t.Fatal("alias")
		}
	}
	for _, header := range []string{"application/json;charset=cp1006", "application/json;charset=utf-16", "application/json;charset=utf-32", "application/json;charset=\"broken"} {
		if _, err := Normalize([]byte("{}"), http.Header{"Content-Type": []string{header}}); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(header, err)
		}
	}
	// Explicit endian codecs do not remove their decoded U+FEFF, unlike BOM codecs.
	out, err := Normalize(encoded("{}", 2, binary.LittleEndian, true), http.Header{"Content-Type": []string{"application/json;charset=utf-16le"}})
	if err != nil || string(out) != "\ufeff{}" {
		t.Fatal(string(out), err)
	}
	// Python replacement behavior for unpaired/truncated code units stays visible.
	out, err = Normalize([]byte{0, 0xd8, 0x78}, http.Header{"Content-Type": []string{"application/json;charset=utf-16le"}})
	if err != nil || string(out) != "�" {
		t.Fatal(string(out), err)
	}
	out, err = Normalize([]byte{0, 0, 0x11, 0, 0x41}, http.Header{"Content-Type": []string{"application/json;charset=utf-32le"}})
	if err != nil || string(out) != "��" {
		t.Fatal(string(out), err)
	}
	if out, err := Normalize(nil, http.Header{"Content-Type": []string{"broken; charset=cp1006"}}); err != nil || len(out) != 0 {
		t.Fatal(out, err)
	}
}
