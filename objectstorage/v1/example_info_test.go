package v1_test

import (
	"context"
	"errors"
	"fmt"

	objectstorage "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func reportResponseErrorInfo(err error) {
	var response *resource.ResponseError
	if errors.As(err, &response) {
		fmt.Printf("failed status=%d transaction=%q observed-bytes=%d\n",
			response.StatusCode, response.Header.Get("X-Trans-Id"), len(response.Body))
	}
}

func showSegmentInfo(result *objectstorage.ObjectSegmentSizeResult) {
	if result == nil {
		return
	}
	fmt.Printf("requested=%d max=%d min=%d selected=%d fallback=%t status=%d raw-bytes=%d\n",
		result.RequestedSize, result.MaxFileSize, result.MinSegmentSize,
		result.Size, result.UsedFallback, result.StatusCode, len(result.Body))
}

func capabilitiesInfo(ctx context.Context, service *objectstorage.Service) error {
	info, err := service.GetInfo(ctx,
		objectstorage.WithGetInfoOpts(objectstorage.GetInfoOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		objectstorage.WithGetInfoHeaders(map[string]string{"X-Trace-Label": "capabilitiesInfo"}),
		request.Header("X-Request-Label", "info"),
	)
	if err != nil {
		reportResponseErrorInfo(err)
		return err
	}
	fmt.Printf("info status=%d transaction=%q\n", info.StatusCode, info.Header.Get("X-Trans-Id"))
	if info.Swift != nil {
		if maximum, ok := info.Swift["max_file_size"]; ok {
			fmt.Println("advertised max_file_size:", string(maximum))
		}
	}
	fmt.Println("SLO section present:", info.SLO != nil)
	if plugin, ok := info.Body["my_plugin"]; ok {
		fmt.Println("raw plugin:", string(plugin))
	}

	large := int64(8 * 1024 * 1024 * 1024)
	defaultOptions := []objectstorage.ObjectSegmentSizeOption{
		objectstorage.WithObjectSegmentSizeOpts(objectstorage.ObjectSegmentSizeOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
			Size:    &large,
		}),
		objectstorage.WithObjectSegmentSizeHeaders(map[string]string{"X-Trace-Label": "segment"}),
		request.Header("X-Request-Label", "default"),
		request.Custom(func(c *objectstorage.ObjectSegmentSizeOpts) error {
			v := int64(2 * 1024 * 1024 * 1024)
			c.Size = &v
			return nil
		}),
		request.Custom(func(c *objectstorage.ObjectSegmentSizeOpts) error { c.Size = nil; return nil }),
	}
	selected, err := service.GetObjectSegmentSize(ctx, defaultOptions...)
	showSegmentInfo(selected)
	if err != nil {
		reportResponseErrorInfo(err)
		return err
	}

	selected, err = service.GetObjectSegmentSize(ctx,
		request.Custom(func(c *objectstorage.ObjectSegmentSizeOpts) error { v := large; c.Size = &v; return nil }),
	)
	showSegmentInfo(selected)
	if err != nil {
		reportResponseErrorInfo(err)
	}
	return err
}

func Example_info() {}
