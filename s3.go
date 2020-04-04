package s3browser

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Client struct {
	s3     *s3.S3
	bucket string
}

func NewS3Client(cfg Config) S3Client {
	conf := aws.NewConfig()
	conf.WithEndpoint(cfg.Endpoint)
	conf.WithRegion(cfg.Region)
	conf.WithDisableSSL(!cfg.Secure)
	conf.WithCredentials(credentials.NewChainCredentials([]credentials.Provider{
		&credentials.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     cfg.Key,
				SecretAccessKey: cfg.Secret,
			},
		},
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{},
	}))

	return S3Client{
		s3:     s3.New(session.New(conf)),
		bucket: cfg.Bucket,
	}
}

func (c *S3Client) ForEachObject(fn func(*s3.Object)) error {
	return c.s3.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket: &c.bucket,
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, obj := range page.Contents {
				fn(obj)
			}
			return true
		},
	)
}
