package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	accessKey  = "4d7611f748dbe2573bdc94bac4efb5cc"
	secretKey  = "39756a642e3fd8ff119efeaffd7fd0d0f8afbd73af4f16b70fba1fcea4283ad2"
	region     = "wnam" // specify your region
	endpoint   = "https://250397b01822ad832478cabd941e8740.r2.cloudflarestorage.com"
	bucketName = "vercel-clone"
	timeout    = 10 * time.Minute
)

var config = &aws.Config{
	Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
	Endpoint:         aws.String(endpoint),
	Region:           aws.String(region),
	S3ForcePathStyle: aws.Bool(true),
}

func DownloadS3Folder(prefix string) error {
	sess := session.Must(session.NewSession(config))
	svc := s3.New(sess)

	result, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return err
	}

	for _, item := range result.Contents {
		key := aws.StringValue(item.Key)
		if key == "" {
			continue
		}

		cwd, err := os.Getwd()
		if err != nil {
			log.Println(err)
			return nil
		}

		finalOutputPath := filepath.Join(cwd, fmt.Sprintf("/output/%v", key))
		dirName := filepath.Dir(finalOutputPath)

		if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
			return err
		}

		file, err := os.Create(finalOutputPath)
		if err != nil {
			return err
		}

		defer file.Close()

		getObjectOutput, err := svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			return err
		}

		_, err = io.Copy(file, getObjectOutput.Body)
		if err != nil {
			return err
		}

		fmt.Printf("Downloaded: %s\n", finalOutputPath)
	}

	return nil
}

func UploadFilesToS3(projectID string) error {

	sess := session.Must(session.NewSession(config))
	svc := s3.New(sess)
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
		return err
	}

	// Create a context with a timeout that will abort the upload if it takes
	// more than the passed in timeout.
	ctx := context.Background()
	var cancelFn func()
	if timeout > 0 {
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
	}
	// Ensure the context is canceled to prevent leaking.
	// See context package for more information, https://golang.org/pkg/context/
	if cancelFn != nil {
		defer cancelFn()
	}

	directory := filepath.Join(cwd, fmt.Sprintf("/output/%v/dist", projectID))

	files, err := GetFilesList(directory)
	if err != nil {
		log.Println(err)
		return err
	}

	for _, file := range files {
		absolutePath := file
		filesPath := string([]rune(file)[len(directory)+1:])
		fileContent, err := os.ReadFile(absolutePath)
		if err != nil {
			log.Println(err)
			return err
		}

		log.Printf("Pushing %v", fmt.Sprintf("%v/dist/%v", projectID, filesPath))

		_, err = svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fmt.Sprintf("%v/dist/%v", projectID, filesPath)),
			Body:   bytes.NewReader(fileContent),
		})

		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

func GetFilesList(dir string) ([]string, error) {
	files := []string{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
