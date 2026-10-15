// Package responsejson provides bounded response-charset normalization for
// Source compatibility readers while callers retain the original HTTP receipt.
package responsejson

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Normalize returns owned UTF-8 JSON bytes for explicit UTF-16/32 and Latin-1
// response charsets. Empty bodies and absent/UTF-8 charsets are cloned unchanged.
// UTF-16/32 without a byte-order suffix require a BOM; explicit LE/BE codecs
// retain a BOM as data, matching Python's explicit-endian codecs. Malformed code
// units use replacement characters. This is not the complete Requests encoding
// detector, codec registry or JSON loader. Callers must preserve body for proof.
func Normalize(body []byte, header http.Header) ([]byte, error) {
	if len(body) == 0 {
		return bytes.Clone(body), nil
	}
	contentType := header.Get("Content-Type")
	if contentType == "" {
		return bytes.Clone(body), nil
	}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, unsupported("malformed Content-Type: %v", err)
	}
	charset := strings.ToLower(strings.TrimSpace(params["charset"]))
	charset = strings.ReplaceAll(charset, "_", "-")
	switch charset {
	case "", "utf-8", "utf8":
		return bytes.Clone(body), nil
	case "iso-8859-1", "iso8859-1", "iso88591", "latin-1", "latin1", "l1", "cp819":
		result := make([]rune, len(body))
		for i, b := range body {
			result[i] = rune(b)
		}
		return []byte(string(result)), nil
	}
	width := 0
	var order binary.ByteOrder
	switch charset {
	case "utf-16", "utf16":
		width = 2
		if bytes.HasPrefix(body, []byte{0xff, 0xfe}) {
			order = binary.LittleEndian
			body = body[2:]
		} else if bytes.HasPrefix(body, []byte{0xfe, 0xff}) {
			order = binary.BigEndian
			body = body[2:]
		}
	case "utf-16le", "utf-16-le", "utf16le":
		width = 2
		order = binary.LittleEndian
	case "utf-16be", "utf-16-be", "utf16be":
		width = 2
		order = binary.BigEndian
	case "utf-32", "utf32":
		width = 4
		if bytes.HasPrefix(body, []byte{0xff, 0xfe, 0, 0}) {
			order = binary.LittleEndian
			body = body[4:]
		} else if bytes.HasPrefix(body, []byte{0, 0, 0xfe, 0xff}) {
			order = binary.BigEndian
			body = body[4:]
		}
	case "utf-32le", "utf-32-le", "utf32le":
		width = 4
		order = binary.LittleEndian
	case "utf-32be", "utf-32-be", "utf32be":
		width = 4
		order = binary.BigEndian
	default:
		return nil, unsupported("response charset %q", charset)
	}
	if order == nil {
		return nil, unsupported("response charset %q requires a BOM", charset)
	}
	result := make([]rune, 0, len(body)/width)
	trailing := len(body)%width != 0
	if width == 2 {
		units := make([]uint16, 0, len(body)/2)
		for i := 0; i+1 < len(body); i += 2 {
			units = append(units, order.Uint16(body[i:i+2]))
		}
		result = utf16.Decode(units)
		// A trailing high surrogate and incomplete following unit form one Python
		// truncated-data replacement span, not two independent errors.
		if trailing && len(units) > 0 && units[len(units)-1] >= 0xd800 && units[len(units)-1] <= 0xdbff {
			trailing = false
		}
	} else {
		for i := 0; i+3 < len(body); i += 4 {
			value := order.Uint32(body[i : i+4])
			r := rune(value)
			if value > utf8.MaxRune || value >= 0xd800 && value <= 0xdfff {
				r = utf8.RuneError
			}
			result = append(result, r)
		}
	}
	if trailing {
		result = append(result, utf8.RuneError)
	}
	return []byte(string(result)), nil
}
func unsupported(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrUnsupported, fmt.Sprintf(format, args...))
}
