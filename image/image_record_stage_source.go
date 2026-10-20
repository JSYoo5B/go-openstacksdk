package image

import (
	"errors"
	"io"
	"syscall"
)

// Infer total seekable length, not remaining bytes. Caller data is never closed
// and a captured cursor is restored even when the end seek fails. ESPIPE means
// an unavailable size, as in Source utils.get_file_size.
func inferImageRecordStageSize(p *preparedImageRecord, data io.Reader) (*int64, error) {
	if data == nil {
		return nil, p.check(p.ctx)
	}
	seeker, seekable := data.(io.Seeker)
	if !seekable {
		return nil, p.check(p.ctx)
	}
	if err := p.check(p.ctx); err != nil {
		return nil, err
	}
	current, currentErr := seeker.Seek(0, io.SeekCurrent)
	currentGuard := p.check(p.ctx)
	if currentErr != nil || currentGuard != nil {
		if currentGuard == nil && errors.Is(currentErr, syscall.ESPIPE) {
			return nil, nil
		}
		return nil, errors.Join(currentErr, currentGuard)
	}
	total, endErr := seeker.Seek(0, io.SeekEnd)
	endGuard := p.check(p.ctx)
	// This is cursor cleanup, not permission to continue a failed operation.
	// Capturing the end-boundary guard first keeps source failures sticky even
	// when a caller restore hook puts the original service binding back.
	_, restoreErr := seeker.Seek(current, io.SeekStart)
	restoreGuard := p.check(p.ctx)
	if endGuard != nil || restoreGuard != nil {
		return nil, errors.Join(endErr, endGuard, restoreErr, restoreGuard)
	}
	if endErr != nil {
		if errors.Is(endErr, syscall.ESPIPE) {
			if restoreErr == nil || errors.Is(restoreErr, syscall.ESPIPE) {
				return nil, nil
			}
		}
		return nil, errors.Join(endErr, restoreErr)
	}
	if restoreErr != nil {
		if errors.Is(restoreErr, syscall.ESPIPE) {
			return nil, nil
		}
		return nil, restoreErr
	}
	return &total, nil
}
