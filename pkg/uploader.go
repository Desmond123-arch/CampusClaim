package pkg

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type SpacesUploader struct {
	Client     *s3.Client
	Bucket     string
	SpaceURL   string
	Region     string
	UploadPath string
}

func NewSpacesUploader(accessKey, secretKey, region, bucket, endpoint string) *SpacesUploader {
	// Ensure endpoint includes https://
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           endpoint,
			SigningRegion: region,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &SpacesUploader{
		Client:   client,
		Bucket:   bucket,
		SpaceURL: endpoint,
		Region:   region,
	}
}

func (u *SpacesUploader) UploadFile(file multipart.File, fileHeade *multipart.FileHeader) (string, error) {
	defer file.Close()

	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(file)
	if err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), fileHeade.Filename)
	_, err = u.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.Bucket),
		Key: aws.String(fileName),
		Body: bytes.NewReader(buf.Bytes()),
		ACL: "public-read",
		ContentType: aws.String(fileHeade.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", err
	}
	publicURL := fmt.Sprintf("%s/%s/%s", u.SpaceURL, u.Bucket, fileName)
	return publicURL, nil
}