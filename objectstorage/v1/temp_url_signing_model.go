package v1

// TempURLDigest chooses the signing hash; the server controls which hashes it accepts.
type TempURLDigest string

const (
	TempURLDigestSHA1   TempURLDigest = "sha1"
	TempURLDigestSHA256 TempURLDigest = "sha256"
	TempURLDigestSHA512 TempURLDigest = "sha512"
)

// FormSignatureInput describes the limits and literal fields for a FormPost signature.
type FormSignatureInput struct {
	ObjectPrefix   string
	RedirectURL    string
	MaxFileSize    int64
	MaxUploadCount int64
	Timeout        int64
}

// FormSignatureResult contains signed form fields and any actual key-discovery evidence.
type FormSignatureResult struct {
	Path      string
	URL       string
	Signature string
	Expires   int64
	Digest    TempURLDigest
	Discovery *TempURLKeyResult
}

// TempURLResult contains an escaped relative URL and any actual key-discovery evidence.
type TempURLResult struct {
	Path      string
	URL       string
	Signature string
	Expires   int64
	Digest    TempURLDigest
	Discovery *TempURLKeyResult
}
