package pkg

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"time"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func UploadFile(endpoint, region, bucket, key, secret string, file multipart.File,fileHeader *multipart.FileHeader ,ctx context.Context) (string, error) {
	s3Config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:    aws.String(endpoint),
		Region:      aws.String(region),
		// S3ForcePathStyle: aws.Bool(false),
	}
	sess, err := session.NewSession(s3Config)
	if err != nil {
		return "", err
	}
	uploader := s3manager.NewUploader(sess)
	output, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileHeader.Filename),
		Body:   file,
		ACL:    aws.String("public-read"),
		ContentType: aws.String(fileHeader.Header.Get("Content-Type")),
	})

	if err != nil {
		return "", err
	}
	fmt.Println(output)
	return output.Location, err
}

func UploadAsyncSave(file multipart.File, fileHeader *multipart.FileHeader, owner_id uint, uploadType string) error {
	var img models.Images
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	access_key := os.Getenv("DIGITAL_OCEAN_ACCESS")
	secret := os.Getenv("DIGITAL_OCEAN_SECRET")
	bucket := os.Getenv("DIGITAL_OCEAN_BUCKET")
	endpoint := os.Getenv("DIGITAL_OCEAN_ENDPOINT")
	region := os.Getenv("DIGITAL_OCEAN_REGION")

	if uploadType == "profile" {
		endpoint = fmt.Sprintf("%s/profiles", endpoint)
	} else {
		endpoint = fmt.Sprintf("%s/items", endpoint)
	}
	defer cancel()
	fileHeader.Filename = fmt.Sprintf("%d-%s", owner_id, fileHeader.Filename)
	url, err := UploadFile(endpoint, region, bucket, access_key, secret, file, fileHeader,ctx)
	if err != nil {
		return err
	}
	if uploadType == "profile" {
		//DO AN UPDATE HERE
		if err := models.DB.Model(&models.User{}).
		Where("uuid = ?", owner_id).
		Update("profile_image", url).Error; err != nil {
			return err
		}
	} else {
		img = models.Images{
			ImageUrl:  url,
			UpdatedAt: time.Now(),
			ItemID:    owner_id,
		}
		if err := models.DB.Create(&img).Error; err != nil {
			return err
		}
	}
	return nil
}
