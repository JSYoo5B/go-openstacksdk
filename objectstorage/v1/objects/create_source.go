package objects

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

type objectCreateSource struct {
	data []byte
	// Only the concrete image-import profile borrows a single-attempt reader.
	reader io.Reader
	file   *os.File
	path   string
	size   int64
}

func (s *objectCreateSource) section(offset, size int64) io.Reader {
	if s.reader != nil {
		return s.reader
	}
	if s.file != nil {
		return io.NewSectionReader(s.file, offset, size)
	}
	return bytes.NewReader(s.data[offset : offset+size])
}
func (s *objectCreateSource) close() error {
	if s.file == nil {
		return nil
	}
	return joinMetadataErrors(s.file.Close(), os.Remove(s.path))
}

type objectCreateGuardReader struct {
	reader io.Reader
	p      *preparedCreateObject
	ctx    context.Context
}

func (r objectCreateGuardReader) Read(data []byte) (int, error) {
	if err := r.p.guard(r.ctx); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(data)
	return n, joinMetadataErrors(err, r.p.note(r.ctx))
}
func (p *preparedCreateObject) ownInput(ctx context.Context, input CreateObjectInput, kind string) (source *objectCreateSource, err error) {
	if err = p.guard(ctx); err != nil {
		return nil, err
	}
	if kind == "bytes" {
		return &objectCreateSource{data: input.Data, size: int64(len(input.Data))}, nil
	}
	reader := input.Reader
	var original *os.File
	if kind == "file" {
		original, err = os.Open(input.Filename)
		if err != nil {
			return nil, metadataContextError(ctx, err)
		}
		reader = original
	}
	var owned *objectCreateSource
	defer func() {
		if original != nil {
			err = joinMetadataErrors(err, original.Close(), p.note(ctx))
		}
		if err != nil && owned != nil {
			err = joinMetadataErrors(err, owned.close())
			source = nil
		}
	}()
	file, err := os.CreateTemp("", "go-openstacksdk-swift-")
	if err != nil {
		return nil, metadataContextError(ctx, err)
	}
	owned = &objectCreateSource{file: file, path: file.Name()}
	owned.size, err = io.CopyBuffer(file, objectCreateGuardReader{reader: reader, p: p, ctx: ctx}, make([]byte, 32768))
	err = metadataContextError(ctx, joinMetadataErrors(err, p.guard(ctx)))
	if err != nil {
		return nil, err
	}
	return owned, nil
}
func (p *preparedCreateObject) hashReader(ctx context.Context, reader io.Reader) (string, string, error) {
	md, sh := md5.New(), sha256.New()
	_, err := io.CopyBuffer(io.MultiWriter(md, sh), objectCreateGuardReader{reader: reader, p: p, ctx: ctx}, make([]byte, 32768))
	if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
		return "", "", metadataContextError(ctx, err)
	}
	return fmt.Sprintf("%x", md.Sum(nil)), fmt.Sprintf("%x", sh.Sum(nil)), nil
}
func (p *preparedCreateObject) hashFilename(ctx context.Context, filename string) (md, sh string, err error) {
	if err = p.guard(ctx); err != nil {
		return "", "", err
	}
	file, err := os.Open(filename)
	if err != nil {
		return "", "", metadataContextError(ctx, err)
	}
	md, sh, err = p.hashReader(ctx, file)
	err = joinMetadataErrors(err, file.Close(), p.note(ctx))
	if err != nil {
		return "", "", metadataContextError(ctx, err)
	}
	return md, sh, nil
}
