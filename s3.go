package s3browser

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v6"
)

type S3Client struct {
	s3     *minio.Client
	bucket string
}

func NewS3Client(cfg Config) S3Client {
	minioClient, err := minio.New(cfg.Endpoint, cfg.Key, cfg.Secret, cfg.Secure)
	if err != nil {
		log.Fatalln(err)
	}

	return S3Client{
		s3:     minioClient,
		bucket: cfg.Bucket,
	}
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
	objectOptions.Header().Set("Range", rangeHdr)
	coreClient := minio.Core{Client: c.s3}
	return coreClient.GetObject(c.bucket, filePath, objectOptions)
}
