package image

import (
	"bytes"
	"context"
	"errors"
	"io"
	"unicode/utf8"
)

type imageRecordCreateDataKind uint8

const (
	imageRecordCreateDataNone imageRecordCreateDataKind = iota
	imageRecordCreateDataBytes
	imageRecordCreateDataText
	imageRecordCreateDataReader
)

// ImageRecordCreateData preserves None, empty binary/text values and a borrowed
// reader as distinct Source values. Use its concrete factories; capture never
// reads, seeks or closes a reader. The zero value represents None.
type ImageRecordCreateData struct {
	kind       imageRecordCreateDataKind
	data       []byte
	reader     io.Reader
	captureErr error
}

// ImageRecordCreateBytes owns a snapshot, including a present empty nil slice.
func ImageRecordCreateBytes(value []byte) ImageRecordCreateData {
	return ImageRecordCreateData{kind: imageRecordCreateDataBytes, data: bytes.Clone(value)}
}

// ImageRecordCreateText owns UTF-8 text. Empty text remains present but falsey.
func ImageRecordCreateText(value string) ImageRecordCreateData {
	data := ImageRecordCreateData{kind: imageRecordCreateDataText, data: []byte(value)}
	if !utf8.ValidString(value) {
		data.captureErr = uploadInvalid("image create text must be UTF-8")
	}
	return data
}

// ImageRecordCreateReader borrows a reader at its current cursor. A nil reader
// represents None; typed nil is retained as an error until operation capture.
func ImageRecordCreateReader(value io.Reader) ImageRecordCreateData {
	if value == nil {
		return ImageRecordCreateData{}
	}
	data := ImageRecordCreateData{kind: imageRecordCreateDataReader, reader: value}
	if isNilReader(value) {
		data.captureErr = uploadInvalid("image create reader must not be typed nil")
	}
	return data
}

func captureImageRecordCreateData(ctx context.Context, check func(context.Context) error, value ImageRecordCreateData) (ImageRecordCreateData, error) {
	if err := imageRecordImportCheck(ctx, check); err != nil {
		return value, err
	}
	value.data = bytes.Clone(value.data)
	if value.kind > imageRecordCreateDataReader {
		value.captureErr = errors.Join(value.captureErr, uploadInvalid("invalid image create data kind"))
	}
	if value.kind == imageRecordCreateDataReader && (value.reader == nil || isNilReader(value.reader)) {
		value.captureErr = errors.Join(value.captureErr, uploadInvalid("image create reader must not be typed nil"))
	}
	return value, errors.Join(value.captureErr, imageRecordImportCheck(ctx, check))
}
func (value ImageRecordCreateData) present() bool { return value.kind != imageRecordCreateDataNone }
func (value ImageRecordCreateData) truthy() bool {
	if value.kind == imageRecordCreateDataReader {
		return true
	}
	return len(value.data) != 0
}
func (value ImageRecordCreateData) openReader() io.Reader {
	switch value.kind {
	case imageRecordCreateDataBytes, imageRecordCreateDataText:
		return bytes.NewReader(value.data)
	case imageRecordCreateDataReader:
		return value.reader
	default:
		return nil
	}
}
