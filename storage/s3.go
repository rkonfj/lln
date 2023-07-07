package storage

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rkonfj/lln/config"
)

func S3SignRequest(namespace, filepath string) (url string, err error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			config.Conf.Storage.S3.AccessKeyID,
			config.Conf.Storage.S3.SecretAccessKey, ""),
		Endpoint: &config.Conf.Storage.S3.Endpoint,
		Region:   aws.String(config.Conf.Storage.S3.Region),
	})
	if err != nil {
		return
	}

	svc := s3.New(sess)

	req, _ := svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(config.Conf.Storage.S3.Bucket),
		Key:    aws.String(fmt.Sprintf("/%s/%s", namespace, filepath)),
	})
	url, _, err = req.PresignRequest(15 * time.Minute)
	if err != nil {
		return
	}
	return
}
