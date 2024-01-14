package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var BUCKET_NAME = "bucket-name"
var USE_SSL = false
var BATCH_SIZE = 1024

type MinioAccessor struct {
  dockerClient *client.Client
	minioClients   map[MinioInstance]*minio.Client
}

func InitMinioClient() MinioAccessor {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
  return MinioAccessor{dockerClient: cli, minioClients: make(map[MinioInstance]*minio.Client)}
}

func (client *MinioAccessor) getMinioClientForInstance(instance MinioInstance) (*minio.Client, error) {
	minioClient, err := minio.New(instance.host, &minio.Options{
		Creds:  credentials.NewStaticV4(instance.accessKey, instance.secretKey, ""),
		Secure: USE_SSL,
	})
	if err != nil {
		return nil, err
	}
  return minioClient, nil
}

// What should happen for duplicated object ids?
func (client *MinioAccessor) sendContentToMinioInstance(objectName string, instance MinioInstance, reader io.Reader, objectSize int64) error {
	ctx := context.Background()

	// Initialize minio client object.
  minioClient, err := client.getMinioClientForInstance(instance)
  if err != nil {
    return err
  }

	// create a bucket if not exists
	err = minioClient.MakeBucket(ctx, BUCKET_NAME, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, BUCKET_NAME)
		if errBucketExists == nil && exists {
			fmt.Printf("Bucket exists: %s\n", BUCKET_NAME)
		} else {
			return err
		}
	}

	// upload the file with content from request
	fileSize := -1 // FIXME: this should be the actual size of body. Maybe this is passed in request somewhere?
	_, err = minioClient.PutObject(ctx, BUCKET_NAME, objectName, reader, int64(fileSize), minio.PutObjectOptions{})
	if err != nil {
		// if this is size mismatch we can live with this - the element is written to minio instance.
		fmt.Println(err.Error())
	}
	return nil
	// TODO: consider verifying if the file was uploaded successfully
}

func (client *MinioAccessor) getMinioInstancesInfo() (instances []MinioInstance, err error) {
	ctx := context.Background()
	containers, err := client.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		if container.Image != "minio/minio" {
			fmt.Printf("Skipping container %s of image %s\n", container.ID, container.Image)
			continue
		}
    instance, err := client.getMinioInstanceImportantSecrets(ctx, container.ID)
    if err != nil {
      break
    }
		instances = append(instances, *instance)
	}
  return
}

func (client *MinioAccessor) getMinioInstanceImportantSecrets(ctx context.Context,  containerId string) (*MinioInstance, error) {
	json, err := client.dockerClient.ContainerInspect(ctx, containerId)
	if err != nil {
    return nil, err
	}
	secrets := MinioInstance{}
	accessKeyPrefix := "MINIO_ACCESS_KEY="
	secretKeyPrefix := "MINIO_SECRET_KEY="
	minioApiPort := ":9000"
	// Find the IPAddress
	for _, settings := range json.NetworkSettings.Networks {
		// Can I match the network by the name? There should be only one anyways.
		secrets.host = settings.IPAddress + minioApiPort
	}
	// Find secrets from environment variables
	for _, envVar := range json.Config.Env {
		if secrets.accessKey != "" && secrets.secretKey != "" {
			// we're good, found both access and secret key
			break
		}
		if strings.HasPrefix(envVar, accessKeyPrefix) {
			// this is the access key after the equation mark
			secrets.accessKey = envVar[len(accessKeyPrefix):]
			continue
		}
		if strings.HasPrefix(envVar, secretKeyPrefix) {
			// this is the secret key after the equation mark
			secrets.secretKey = envVar[len(accessKeyPrefix):]
			continue
		}
	}
	return &secrets, nil
}

func (client *MinioAccessor) getMinioInstanceObject(objectName string, instance MinioInstance) ([]byte, error) {
	ctx := context.Background()

	// Initialize minio client object.
  minioClient, err := client.getMinioClientForInstance(instance)
  if err != nil {
    return nil, err
  }
	minioObject, err := minioClient.GetObject(ctx, BUCKET_NAME, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
  // Pass the file content in batches onto the response.
  response := []byte{}
  for {
    fileBody := make([]byte, BATCH_SIZE)
    _, err = minioObject.Read(fileBody)
    if err == io.EOF {
      break
    }
    response = append(response, fileBody...)
  }
  return response, nil
}
