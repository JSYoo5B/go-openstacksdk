package v1

import (
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

// TempURLKeyResult retains only the metadata responses actually accepted.
// A nil Key means neither scope supplied a usable nonempty key.
type TempURLKeyResult struct {
	Key           []byte
	FromContainer bool
	Secondary     bool
	Container     *containers.GetMetadataResult
	Account       *accounts.GetMetadataResult
}
