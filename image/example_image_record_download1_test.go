package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type preferencesImageRecordDownload1 []string

func (value *preferencesImageRecordDownload1) String() string { return fmt.Sprint([]string(*value)) }
func (value *preferencesImageRecordDownload1) Set(store string) error {
	*value = append(*value, store)
	return nil
}

func Example_string() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	id := flag.String("id", "", "required literal image ID")
	mode := flag.String("mode", "file", "file, writer, buffer or stream")
	output := flag.String("output", "", "output path for file, writer or stream")
	viaService := flag.Bool("service", false, "call image.Service instead of Connection")
	chunk := flag.Int("chunk-size", 1<<20, "SDK consumed-mode copy chunk in bytes, up to 64 MiB")
	verify := flag.Bool("verify-checksum", true, "verify available hashes in SDK consumed modes")
	timeout := flag.Duration("timeout", 2*time.Minute, "authentication, GETs and consumption deadline")
	var stores preferencesImageRecordDownload1
	flag.Var(&stores, "prefer", "store preference; repeat to preserve order and duplicates")
	flag.Parse()
	if *id == "" {
		log.Fatal("-id is required by this example")
	}
	if *mode != "file" && *mode != "writer" && *mode != "buffer" && *mode != "stream" {
		log.Fatal("-mode must be file, writer, buffer or stream")
	}
	if *mode != "buffer" && *output == "" {
		log.Fatal("this mode requires -output")
	}
	if *timeout <= 0 {
		log.Fatal("-timeout must be positive")
	}
	options := []image.ImageRecordDownloadOption{
		image.WithImageRecordDownloadOpts(image.ImageRecordDownloadOpts{
			Headers: map[string]string{"X-Request-Source": "image-record-download-example"},
		}),
		request.Custom(func(c *image.ImageRecordDownloadOpts) error { c.Stream = *mode == "stream"; return nil }),
		request.Custom(func(c *image.ImageRecordDownloadOpts) error { v := *chunk; c.ChunkSize = &v; return nil }),
		request.Custom(func(c *image.ImageRecordDownloadOpts) error { v := *verify; c.VerifyChecksum = &v; return nil }),
		image.WithImageRecordDownloadStorePreferences(stores...),
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runImageRecordDownload1(ctx, *cloud, *id, *mode, *output, *viaService, options); err != nil {
		log.Fatal(err)
	}
}

func runImageRecordDownload1(ctx context.Context, cloud, id, mode, output string, viaService bool, options []image.ImageRecordDownloadOption) error {
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	input := image.ImageRecordDownloadRequest{ID: id}
	var writer *os.File
	if mode == "file" {
		input.Filename = output
	} else if mode == "writer" {
		// Caller creates/truncates this file before the SDK metadata GET.
		writer, err = os.Create(output)
		if err != nil {
			return err
		}
		input.Output = writer
	}
	var result *image.ImageRecordDownloadResult
	var operationErr error
	if viaService {
		service, serviceErr := conn.Image(ctx)
		if serviceErr != nil {
			operationErr = serviceErr
		} else {
			result, operationErr = service.DownloadImageRecord(ctx, input, options...)
		}
	} else {
		result, operationErr = conn.DownloadImageRecord(ctx, input, options...)
	}
	if writer != nil {
		operationErr = errors.Join(operationErr, writer.Close())
	}
	var callerBytes int64
	if result != nil && result.Downloaded != nil && result.Downloaded.Stream != nil {
		// Consumption and Close happen while the original context is alive.
		// These caller bytes do not update result.BytesWritten or Checksum.
		var consumeErr error
		callerBytes, consumeErr = saveStreamImageRecordDownload1(result.Downloaded.Stream, output)
		operationErr = errors.Join(operationErr, consumeErr)
	}
	return printDownloadImageRecordDownload1(result, operationErr, callerBytes)
}

func saveStreamImageRecordDownload1(stream io.ReadCloser, filename string) (count int64, err error) {
	defer func() { err = errors.Join(err, stream.Close()) }()
	output, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, output.Close()) }()
	return io.Copy(output, stream)
}

func printDownloadImageRecordDownload1(result *image.ImageRecordDownloadResult, operationErr error, callerBytes int64) error {
	value := map[string]any{
		"partial":                     result != nil && operationErr != nil,
		"caller_stream_bytes_written": callerBytes,
	}
	if result != nil {
		value["bytes_written"] = result.BytesWritten
		if result.Metadata != nil {
			value["metadata"] = map[string]any{
				"body": result.Metadata.Body, "header": result.Metadata.Header,
				"status_code": result.Metadata.StatusCode,
			}
		}
		if result.Record != nil {
			value["record"] = map[string]any{
				"resource": result.Record.Resource, "wire": result.Record.Wire,
				"status_code":    result.Record.StatusCode,
				"import_methods": result.Record.ImportMethods,
			}
		}
		if result.Downloaded != nil {
			value["downloaded"] = map[string]any{
				"buffered_bytes":       len(result.Downloaded.Body),
				"status_code":          result.Downloaded.StatusCode,
				"header":               result.Downloaded.Header,
				"compatibility_header": result.Downloaded.CompatibilityHeader,
				"content_md5":          result.Downloaded.ContentMD5,
			}
		}
		if result.Checksum != nil {
			value["checksum"] = map[string]any{
				"algorithm": result.Checksum.Algorithm,
				"expected":  result.Checksum.Expected, "actual": result.Checksum.Actual,
				"complete": result.Checksum.Complete, "verified": result.Checksum.Verified,
			}
		}
	}
	if operationErr != nil {
		value["error"] = operationErr.Error()
		value["checksum_mismatch"] = errors.Is(operationErr, image.ErrChecksumMismatch)
		value["hash_length_required"] = errors.Is(operationErr, image.ErrImageRecordDownloadHashLengthRequired)
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err == nil {
		_, err = fmt.Fprintln(os.Stdout, string(body))
	}
	return errors.Join(operationErr, err)
}
