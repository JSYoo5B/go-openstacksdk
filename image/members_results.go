package image

import (
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
)

// ImageMemberAcknowledgement retains an actual 204 member-deletion response.
// IDs identify the fixed requested target. Body is opaque and may be partial
// when an accepted response returns a read, Close or context error.
type ImageMemberAcknowledgement struct {
	ImageID    string
	MemberID   string
	Body       []byte
	Header     http.Header
	StatusCode int
}

func imageMemberAcknowledgement(imageID, memberID string, response *rest.Response) *ImageMemberAcknowledgement {
	if response == nil || response.StatusCode != http.StatusNoContent {
		return nil
	}
	return &ImageMemberAcknowledgement{ImageID: imageID, MemberID: memberID,
		Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
