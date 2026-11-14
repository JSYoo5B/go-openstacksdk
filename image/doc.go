// Package image exposes Glance images through shared resource policies and a
// metadata creation, direct upload, and optional active-wait workflow.
// Connection.Image returns its Service.
//
// Service groups the higher-level Glance workflows: uploads (UploadImage,
// Upload, CreateAndImport), reads and downloads (GetImage, ListImages,
// ImageTasks, DownloadTo), updates (UpdateImage, SetImageProperties, AddTag,
// RemoveTag, DeactivateImage, ReactivateImage), deletion (DeleteImage),
// members, locations, cache, schemas and tasks. Each method's documentation
// describes its requests, defaults and returned evidence; the image/v2 packages
// hold the native Glance calls.
package image
