package s3browser

import (
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v6"
)

type S3Client struct {
	s3     *minio.Client
	bucket string
}

func NewS3Client(endpoint, key, secret string, secure bool, bucket string) (S3Client, error) {
	minioClient, err := minio.New(endpoint, key, secret, secure)
	c := S3Client{
		s3:     minioClient,
		bucket: bucket,
	}
	return c, err
}

func (c *S3Client) ForEachObject(fn func(minio.ObjectInfo)) error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	objectCh := c.s3.ListObjectsV2(c.bucket, "", true, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			return object.Err
		}
		fn(object)
	}
	return nil
}

func (c *S3Client) GetObject(filePath string, rangeHdr string) (io.ReadCloser, minio.ObjectInfo, http.Header, error) {
	filePath = strings.TrimLeft(filePath, "/")
	objectOptions := minio.GetObjectOptions{}
	objectOptions.Set("Range", rangeHdr)
	coreClient := minio.Core{Client: c.s3}
	return coreClient.GetObject(c.bucket, filePath, objectOptions)
}
