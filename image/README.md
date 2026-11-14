# Image

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
image = conn.image.find_image("ubuntu", ignore_missing=False)
for image in conn.image.images(status="active"):
    print(image.id, image.name)
```

```go
service, err := conn.Image(ctx)
if err != nil { return err }
image, err := service.Images.Find(ctx, resource.Name("ubuntu"))
if err != nil { return err }
fmt.Println(image.ID, image.SizeBytes)

for image, err := range service.Images.List(ctx, resource.WithStatus("active")) {
    if err != nil { return err }
    fmt.Println(image.ID, image.Name)
}
```

```python
with open("ubuntu.qcow2", "rb") as data:
    uploaded = conn.create_image(
        name="ubuntu",
        data=data,
        disk_format="qcow2",
        container_format="bare",
        visibility="private",
        allow_duplicates=True,
        disable_vendor_agent=False,
        use_import=False,
        wait=True,
        timeout=300,
    )
```

```go
service, err := conn.Image(ctx)
if err != nil { return err }
data, err := os.Open("ubuntu.qcow2")
if err != nil { return err }
defer data.Close()

uploaded, err := service.Upload(ctx, image.UploadImageRequest{
    Name: "ubuntu",
    Data: data,
}, image.WithWait(resource.WithTimeout(5*time.Minute)))
if err != nil {
    if uploaded != nil { fmt.Println("created image:", uploaded.ID) }
    return err
}
fmt.Println(uploaded.ID, uploaded.Status)
```

```go
uploaded, err := service.Upload(ctx, image.UploadImageRequest{
    Name: "arm-image",
    Data: data,
}, image.WithDiskFormat("raw"),
   image.WithMinDisk(0), image.WithMinRAM(512),
   image.WithVisibility(image.VisibilityShared),
   image.WithProtected(false), image.WithHidden(false),
   image.WithTags("linux", "arm64"),
   image.WithProperty("hw_architecture", "aarch64"))
if err != nil { return err }
fmt.Println(uploaded.ID)
```

```go
ready, err := service.Images.Wait(ctx, resource.ID(image.ID), "active")
if err != nil { return err }
fmt.Println(ready.Status)

if err := service.Images.Delete(ctx, resource.ID(image.ID)); err != nil {
    return err
}
```

## References

- [..](../README.md)
- [Delete](delete.md)
- [Image Record Download](image-record-download.md)
- [Download](download.md)
- [Image Record Create](image-record-create.md)
- [Image Record Upload](image-record-upload.md)
- [Image Record Import](image-record-import.md)
- [Image V2](https://docs.openstack.org/openstacksdk/latest/user/proxies/image_v2.html)
- [Finding Identities](../docs/finding-identities.md)
- [Image v2 API](v2/README.md)
- [Create Import](create-import.md)
- [image_test.go](image_test.go)
- [upload_test.go](upload_test.go)
- [upload_retry_test.go](upload_retry_test.go)
- [Server Create Test](../connectiontest/server_create_test.go)
- [Service Waits](../docs/service-waits.md)
- [Tasks](tasks.md)
- [Image Tasks](image-tasks.md)
- [Images](images.md)
- [Image Records](image-records.md)
- [Image Record Cloud](image-record-cloud.md)
- [Image Record Waits](image-record-waits.md)
- [Image Record Update](image-record-update.md)
- [Image Record Properties](image-record-properties.md)
- [Image Record Tags](image-record-tags.md)
- [Update](update.md)
- [Update](v2/images/update.md)
- [Upload Image](upload-image.md)
- [Tasks](v2/tasks/README.md)
- [Imageimport](v2/imageimport/README.md)
- [Image Record Stage](image-record-stage.md)
- [Imagedata](v2/imagedata/README.md)
- [Image Record Delete](image-record-delete.md)
- [Delete](v2/images/delete.md)
- [Image Record Actions](image-record-actions.md)
- [Serviceinfo](v2/serviceinfo/README.md)
- [Records](v2/serviceinfo/records.md)
- [Python Store Details](v2/serviceinfo/python-store-details.md)
- [Usage](v2/serviceinfo/usage.md)
- [Mutations](mutations.md)
- [Locations](locations.md)
- [Schemas](schemas.md)
- [Schema Records](schema-records.md)
- [Cache](cache.md)
- [Members](members.md)
- [Member Record Mutations](member-record-mutations.md)
- [Member Records](member-records.md)
- [Metadefnamespaces](v2/metadefnamespaces/README.md)
- [Metadefobjects](v2/metadefobjects/README.md)
- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Resource Types Records](metadef-resource-types-records.md)
- [Metadef Property Records](metadef-property-records.md)
- [Metadef Property](metadef-property.md)
- [Metadef Object Namespace Record Lists](metadef-object-namespace-record-lists.md)
- [Metadef Resource Type Association Mutations](metadef-resource-type-association-mutations.md)
- [Metadef Property Record List](metadef-property-record-list.md)
- [Metadef Property Record Write](metadef-property-record-write.md)
