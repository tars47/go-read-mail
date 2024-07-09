package awss3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Used when s3 returns NOSuchKey error
const NotFound = "NoSuchKey"

// S3 client
var c *s3.Client

// S3 presign client
var pc *s3.PresignClient

// S3 default bucket used for all users
var bucket = "go-read-mail"

// On app startup init aws config and init s3 clients
func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("[s3 init] err loading aws config: %v\n", err)
	}

	c = s3.NewFromConfig(cfg)
	pc = s3.NewPresignClient(c)
}

// Uploads file to s3
func UploadFile(key string, r io.Reader) (string, error) {
	_, err := c.PutObject(
		context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   r,
		})
	if err != nil {
		return "", fmt.Errorf("couldn't upload file %v. err: %v", key, err)
	}

	// Get pre signed url and return the url
	return GetFileLink(key)
}

// DOwnload file from s3, return pointer to bytes.Buffer
func DownloadFile(key string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	// Get the object
	result, err := c.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	// Read the body
	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("[DownloadFile] Couldn't read file body %v. err: %v\n", key, err)
		return nil, err
	}
	// Write to a buffer
	buf.Write(body)
	// Return address of that buffer
	return &buf, nil
}

// Returns pre signed url
func GetFileLink(key string) (string, error) {
	purl, err := pc.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(time.Hour*168))
	if err != nil {
		return "", fmt.Errorf("couldn't get file url %v. err: %v", key, err)
	}
	return purl.URL, nil
}
