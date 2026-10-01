package cyborg

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/accelerator/v2/common"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func RequireMicroversion(client *gophercloud.ServiceClient, minor int) error {
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: Cyborg service client is required", resource.ErrInvalidOption)
	}
	version := client.Microversion
	if version == "" {
		version = "2.0"
	}
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return fmt.Errorf("%w: invalid Cyborg microversion %q", resource.ErrInvalidOption, version)
	}
	for _, part := range parts {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return fmt.Errorf("%w: invalid Cyborg microversion %q", resource.ErrInvalidOption, version)
		}
	}
	major, err1 := strconv.Atoi(parts[0])
	selected, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("%w: invalid Cyborg microversion %q", resource.ErrInvalidOption, version)
	}
	if major != 2 || selected < minor {
		return fmt.Errorf("%w: Cyborg microversion 2.%d or newer required, selected %s", resource.ErrUnsupported, minor, version)
	}
	return nil
}

func Headers(extra map[string]string) (map[string]string, error) {
	reserved := map[string]string{"X-Auth-Token": "", "Host": "", "Content-Length": "", "Content-Type": "", "Accept": "", "User-Agent": "", "OpenStack-API-Version": ""}
	merged, err := request.MergeHeadersFor(reserved, extra, struct{}{})
	if err != nil {
		return nil, err
	}
	for key := range reserved {
		delete(merged, key)
	}
	return merged, nil
}

func Mutate(ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, codes ...int) (*common.Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	guarded, err := guardedClient(client)
	if err != nil {
		return nil, err
	}
	response, err := guarded.Request(ctx, method, endpoint, &gophercloud.RequestOpts{JSONBody: body, MoreHeaders: headers, OkCodes: codes})
	if err != nil {
		return nil, err
	}
	meta := ResponseMetadata(response)
	return &meta, nil
}

// DecodeMutation retains response metadata through resource decoders.
func DecodeMutation[T any](ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, single string, metadata func(*T) *common.Metadata, codes ...int) (*T, error) {
	return fetch[T](ctx, client, method, endpoint, body, headers, single, metadata, codes...)
}
