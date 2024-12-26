package gcloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	storage "cloud.google.com/go/storage"
	log "github.com/gofiber/fiber/v2/log"

	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
)

type CloudStorageHandler struct {
	client    *storage.Client
	control   *control.StorageControlClient
	projectId string
}

func NewCloudStorageHandler(projectId string) (*CloudStorageHandler, error) {
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorw(
			"failure on creating a new cloud storage client",
			"reason",
			err.Error(),
		)
		return nil, err
	}

	ctx = context.Background()

	controllClient, err := control.NewStorageControlClient(ctx)
	if err != nil {
		log.Errorw(
			"failure on creating a new cloud storage control client",
			"reason",
			err.Error(),
		)
		return nil, err
	}

	csh := CloudStorageHandler{
		projectId: projectId,
		client:    client,
		control:   controllClient,
	}

	return &csh, nil
}

func (csb *CloudStorageHandler) CreateBucket(ctx context.Context, bucketName string) (
	bucket *storage.BucketHandle,
	err error,
) {
	if strings.Contains(bucketName, "projects/312193984213/buckets") {
		log.Warnw("Attempt to create a bucket with invalid name", "attempt", bucketName)
		return nil, errors.New("Bad bucket name was given")
	}
	bucket = csb.client.Bucket(bucketName)

	bucketAttrs := storage.BucketAttrs{
		PublicAccessPrevention: storage.PublicAccessPreventionEnforced,
		Location:               "europe-west2",
		HierarchicalNamespace:  &storage.HierarchicalNamespace{Enabled: true},
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: true,
		},
	}

	err = bucket.Create(
		ctx,
		csb.projectId,
		&bucketAttrs,
	)
	if err != nil {
		log.Errorw(
			"Error occurred when creating a bucket",
			"bucket_name",
			bucketName,
			"reason",
			err.Error(),
		)
	}

	return
}

func (cs *CloudStorageHandler) CreateBucketFolder(
	bucketHandle *storage.BucketHandle,
	folderName string,
) (folder *controlpb.Folder, err error) {
	folderLocal := fmt.Sprintf("projects/_/buckets/%s", bucketHandle.BucketName())

	req := &controlpb.CreateFolderRequest{
		Parent:   folderLocal,
		FolderId: folderName,
	}

	ctx := context.Background()

	newFolder, err := cs.control.CreateFolder(ctx, req)
	if err != nil {
		log.Errorw(
			"Failure on creating a folder on a bucekt",
			"bucket",
			bucketHandle.BucketName(),
			"folder_name",
			folderName,
			"reason",
			err.Error(),
		)
	}

	return newFolder, err
}

type ObjectUploadRequest interface {
	Bucket() string
	Folder() string
	ObjName() string
	ContentType() string
	ContentReader() io.Reader
}

func (cs *CloudStorageHandler) UploadObjectToBucket(
	ctx context.Context,
	log log.CommonLogger,
	uploadReq ObjectUploadRequest,
) error {
	log.Infow(
		"Uploading object to bucket",
		"folder",
		uploadReq.Folder(),
		"name",
		uploadReq.ObjName(),
		"bucket",
		uploadReq.Bucket(),
	)

	objectPath := fmt.Sprintf("%s/%s", uploadReq.Folder(), uploadReq.ObjName())

	object := cs.client.Bucket(uploadReq.Bucket()).Object(objectPath)
	object = object.If(storage.Conditions{DoesNotExist: true})

	wc := object.NewWriter(ctx)
	wc.ChunkSize = 512

	if _, err := io.Copy(wc, uploadReq.ContentReader()); err != nil {
		log.Errorw("Failed to copy data from buffer into Object writer", "reason", err.Error())
		return err
	}

	if err := wc.Close(); err != nil {
		log.Errorw("Error closing the object content writer", "reason", err.Error())
		return err
	}

	log.Infow("Uploaded object data to bucket")

	return nil
}
