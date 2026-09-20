package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generatePresignedURL(
	s3Client *s3.Client,
	bucket,
	key string,
	expTime time.Duration,
) (uri string, err error) {
	client := s3.NewPresignClient(s3Client)

	res, err := client.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(expTime),
	)
	if err != nil {
		return
	}

	return res.URL, nil
}
