package image

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	defaultDownloadChunkSize = 1024 * 1024
	maxDownloadChunkSize     = 64 * 1024 * 1024
)

// DownloadImageOpts configures a full-image transfer to a borrowed writer.
// Nil ChunkSize selects 1 MiB; nil VerifyChecksum verifies available hashes.
// Nil or empty StorePreferences leaves store selection to the server.
type DownloadImageOpts struct {
	ChunkSize        *int
	StorePreferences []string
	VerifyChecksum   *bool
}

type DownloadImageOption func(*DownloadImageOpts) error

// WithDownloadImageOpts snapshots and replaces the complete download policy.
func WithDownloadImageOpts(value DownloadImageOpts) DownloadImageOption {
	snapshot := copyDownloadImageOpts(value)
	return func(config *DownloadImageOpts) error { *config = copyDownloadImageOpts(snapshot); return nil }
}

// WithDownloadChunkSize selects a positive copy buffer size, up to 64 MiB.
func WithDownloadChunkSize(value int) DownloadImageOption {
	return func(config *DownloadImageOpts) error { owned := value; config.ChunkSize = &owned; return nil }
}

// WithDownloadStorePreferences preserves the caller's order and repetitions.
func WithDownloadStorePreferences(values ...string) DownloadImageOption {
	snapshot := append(make([]string, 0, len(values)), values...)
	return func(config *DownloadImageOpts) error {
		config.StorePreferences = append(make([]string, 0, len(snapshot)), snapshot...)
		return nil
	}
}

func WithDownloadChecksumVerification(value bool) DownloadImageOption {
	return func(config *DownloadImageOpts) error { owned := value; config.VerifyChecksum = &owned; return nil }
}

func copyDownloadImageOpts(value DownloadImageOpts) DownloadImageOpts {
	value.ChunkSize = copyCreateImportPointer(value.ChunkSize)
	value.VerifyChecksum = copyCreateImportPointer(value.VerifyChecksum)
	if value.StorePreferences != nil {
		value.StorePreferences = append(make([]string, 0, len(value.StorePreferences)), value.StorePreferences...)
	}
	return value
}

func parseDownloadImageOptions(options []DownloadImageOption) (DownloadImageOpts, error) {
	var config DownloadImageOpts
	for _, apply := range options {
		if apply == nil {
			return config, uploadInvalid("nil download option")
		}
		if err := apply(&config); err != nil {
			return config, err
		}
	}
	// A custom callback may retain config; only the owned copy reaches HTTP.
	snapshot := copyDownloadImageOpts(config)
	if snapshot.ChunkSize == nil {
		value := defaultDownloadChunkSize
		snapshot.ChunkSize = &value
	}
	if *snapshot.ChunkSize <= 0 || *snapshot.ChunkSize > maxDownloadChunkSize {
		return snapshot, uploadInvalid("download chunk size must be between 1 byte and 64 MiB")
	}
	if snapshot.VerifyChecksum == nil {
		value := true
		snapshot.VerifyChecksum = &value
	}
	for _, store := range snapshot.StorePreferences {
		if strings.TrimSpace(store) == "" || !utf8.ValidString(store) || strings.ContainsRune(store, ',') {
			return snapshot, uploadInvalid("download store preference must be a nonempty valid UTF-8 identifier without commas")
		}
		for _, char := range store {
			if unicode.IsControl(char) {
				return snapshot, uploadInvalid("download store preference must not contain controls")
			}
		}
	}
	return snapshot, nil
}
