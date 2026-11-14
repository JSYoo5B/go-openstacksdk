package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
)

type createImageImageRecordCreate func(context.Context, image.ImageRecordCreateRequest, ...image.ImageRecordCreateOption) (*image.ImageRecordCreateResult, error)

func Example_imageRecordCreate() {
	cloud := flag.String("cloud", "", "clouds.yaml entry; omission uses OS_CLOUD/OS_* settings")
	scenario := flag.String("scenario", "directImageRecordCreate", "directImageRecordCreate, bytes, import, taskImageRecordCreate, or metadataImageRecordCreate")
	filename := flag.String("file", "", "local image file for directImageRecordCreate/bytes/import/taskImageRecordCreate")
	name := flag.String("name", "image-example", "image name")
	viaService := flag.Bool("service", false, "call image.Service instead of Connection")
	wait := flag.Bool("wait", true, "wait for Task completion in the taskImageRecordCreate scenario")
	forceTask := flag.Bool("force-taskImageRecordCreate-policy", false, "replace cloud image policy with explicit Task policy")
	flag.Parse()
	if *scenario != "metadataImageRecordCreate" && *filename == "" {
		log.Fatal("this scenario requires -file")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := runImageRecordCreate(ctx, *cloud, *scenario, *filename, *name, *viaService, *wait, *forceTask); err != nil {
		log.Fatal(err)
	}
}

func runImageRecordCreate(ctx context.Context, cloud, scenario, filename, name string, viaService, wait, forceTask bool) error {
	options := []openstack.ConnectionOption{}
	if cloud != "" {
		options = append(options, openstack.WithCloud(cloud))
	}
	if forceTask {
		options = append(options, openstack.WithImageCreatePolicy(
			image.WithImageCreatePolicyTasks(true),
			image.WithImageCreatePolicyObjectStoreEnabled(true),
		))
	}
	conn, err := openstack.Connect(ctx, options...)
	if err != nil {
		return err
	}
	var create createImageImageRecordCreate = conn.CreateImageRecord
	if viaService {
		service, serviceErr := conn.Image(ctx)
		if serviceErr != nil {
			return serviceErr
		}
		create = service.CreateImageRecord
	}
	var result *image.ImageRecordCreateResult
	switch scenario {
	case "directImageRecordCreate":
		result, err = directImageRecordCreate(ctx, create, filename, name)
	case "bytes":
		result, err = validatedBytesImageRecordCreate(ctx, create, filename, name)
	case "import":
		result, err = importedImageRecordCreate(ctx, create, filename, name)
	case "taskImageRecordCreate":
		result, err = taskImageRecordCreate(ctx, create, filename, name, wait)
	case "metadataImageRecordCreate":
		result, err = metadataImageRecordCreate(ctx, create, name)
	default:
		return fmt.Errorf("unknown scenario %q", scenario)
	}
	return showImageRecordCreate(result, err)
}

func directImageRecordCreate(ctx context.Context, create createImageImageRecordCreate, filename, name string) (*image.ImageRecordCreateResult, error) {
	return create(ctx, image.ImageRecordCreateRequest{
		Name: name, Filename: filename, Attributes: map[string]any{"visibility": "private"},
	}, image.WithImageRecordCreateAllowDuplicates(true))
}

func validatedBytesImageRecordCreate(ctx context.Context, create createImageImageRecordCreate, filename, name string) (*image.ImageRecordCreateResult, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return create(ctx, image.ImageRecordCreateRequest{
		Name: name, Data: image.ImageRecordCreateBytes(data),
	}, image.WithImageRecordCreateValidateChecksum(true))
}

func importedImageRecordCreate(ctx context.Context, create createImageImageRecordCreate, filename, name string) (*image.ImageRecordCreateResult, error) {
	return create(ctx, image.ImageRecordCreateRequest{Name: name, Filename: filename},
		image.WithImageRecordCreateAllowDuplicates(true),
		image.WithImageRecordCreateUseImport(true),
		image.WithImageRecordCreateImportOptions(image.WithImageRecordImportMethod("glance-directImageRecordCreate")),
	)
}

func taskImageRecordCreate(ctx context.Context, create createImageImageRecordCreate, filename, name string, wait bool) (*image.ImageRecordCreateResult, error) {
	return create(ctx, image.ImageRecordCreateRequest{Name: name, Filename: filename},
		image.WithImageRecordCreateAllowDuplicates(true),
		image.WithImageRecordCreateContainer("images"),
		image.WithImageRecordCreateWait(wait),
		image.WithImageRecordCreateTimeout(300),
	)
}

func metadataImageRecordCreate(ctx context.Context, create createImageImageRecordCreate, name string) (*image.ImageRecordCreateResult, error) {
	return create(ctx, image.ImageRecordCreateRequest{
		Name: name, Attributes: map[string]any{"min_disk": "0", "hw_qemu_guest_agent": false},
	}, image.WithImageRecordCreateAllowDuplicates(true),
		image.WithImageRecordCreateTags([]string{"sdk-example"}),
		image.WithImageRecordCreateMeta(map[string]any{"vendor_raw": map[string]any{"enabled": false}}),
	)
}

func recordSummaryImageRecordCreate(record *image.ImageRecord) any {
	if record == nil {
		return nil
	}
	return map[string]any{
		"resource": record.Resource, "wire": record.Wire,
		"status_code": record.StatusCode, "header": record.Header,
		"envelope_bytes": []byte(record.Envelope), "import_methods": record.ImportMethods,
	}
}

func taskSummaryImageRecordCreate(value *image.ImageRecordCreateTask) any {
	if value == nil {
		return nil
	}
	return map[string]any{
		"resource": value.Resource, "status_code": value.StatusCode,
		"header": value.Header, "envelope_bytes": []byte(value.Envelope),
	}
}

func showImageRecordCreate(result *image.ImageRecordCreateResult, operationErr error) error {
	value := map[string]any{"partial": result != nil && operationErr != nil}
	if operationErr != nil {
		value["error"] = operationErr.Error()
	}
	if result != nil {
		value["outcome"], value["reused"], value["warnings"] = result.Outcome, result.Reused, result.Warnings
		value["record"], value["updated"] = recordSummaryImageRecordCreate(result.Record), recordSummaryImageRecordCreate(result.Updated)
		value["taskImageRecordCreate"], value["task_diagnostic"] = taskSummaryImageRecordCreate(result.Task), taskSummaryImageRecordCreate(result.TaskDiagnostic)
		value["checksum"], value["importedImageRecordCreate"] = result.Checksum, result.Imported
		for phase, receipt := range map[string]*image.ImageUploadResponse{
			"created": result.Created, "uploaded": result.Uploaded, "staged": result.Staged,
			"checksum_fetched": result.ChecksumFetched, "task_image_fetched": result.TaskImageFetched,
			"task_diagnostic_response": result.TaskDiagnosticResponse,
		} {
			if receipt != nil {
				value[phase] = receipt
			}
		}
		if result.TaskWait != nil {
			value["task_wait"] = map[string]any{
				"taskImageRecordCreate": taskSummaryImageRecordCreate(result.TaskWait.Task), "original_task": taskSummaryImageRecordCreate(result.TaskWait.OriginalTask),
				"fetches": result.TaskWait.Fetches, "recreations": result.TaskWait.Recreations,
				"fetch_responses": result.TaskWait.FetchResponses, "recreation_responses": result.TaskWait.RecreationResponses,
			}
		}
		if result.Swift != nil {
			value["swift"] = map[string]any{
				"container_discovery": result.Swift.ContainerDiscovery,
				"container_created":   result.Swift.ContainerCreated,
				"container_fetched":   result.Swift.ContainerFetched,
				"object":              result.Swift.Object, "cleanup": result.Swift.Cleanup,
			}
		}
		if result.Cleanup != nil {
			value["image_cleanup_ack"] = result.Cleanup.Acknowledgement
		}
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return errors.Join(operationErr, err)
	}
	fmt.Println(string(body))
	return operationErr
}
