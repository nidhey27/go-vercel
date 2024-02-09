package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"

	"github.com/robinjoseph08/redisqueue"
)

const (
	accessKey  = "215c3c1135fb758f1562d53cd8b332e7"
	secretKey  = "77f268589d6121d7fe43fb21171385382d1d0049caa92436d5fad227abbbfd84"
	region     = "wnam" // specify your region
	endpoint   = "https://250397b01822ad832478cabd941e8740.r2.cloudflarestorage.com"
	bucketName = "vercel-clone"
	timeout    = 10 * time.Minute
)

type Request struct {
	ProjectURL string `json:"project_url"`
}

func Deploy(c *gin.Context) {
	var body Request
	if err := c.BindJSON(&body); err != nil {
		log.Panicln(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"message": "",
			"data":    make([]interface{}, 0),
		})
		return
	}

	// Get Project URL from Request Body
	projectURL := body.ProjectURL

	// Generate a Unique 6 digit ID
	projectId := GenerateID()

	// Clone Github Repository to /output/{id}
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "",
			"data":    make([]interface{}, 0),
		})
		return
	}
	directory := filepath.Join(cwd, fmt.Sprintf("/output/%v", projectId))
	_, err = git.PlainClone(directory, false, &git.CloneOptions{
		URL:               projectURL,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "",
			"data":    make([]interface{}, 0),
		})
		return
	}

	// Get list of all files in the /output/{id}
	files, err := GetFilesList(directory)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "",
			"data":    make([]interface{}, 0),
		})
		return
	}

	// Upload to S3
	config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
	}

	sess := session.Must(session.NewSession(config))
	svc := s3.New(sess)

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

	for _, file := range files {
		absolutePath := file
		filesPath := string([]rune(file)[len(directory)+1:])
		fileContent, err := os.ReadFile(absolutePath)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   err.Error(),
				"message": "",
				"data":    make([]interface{}, 0),
			})
			return
		}
		_, err = svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fmt.Sprintf("%v/%v", projectId, filesPath)),
			Body:   bytes.NewReader(fileContent),
		})

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   err.Error(),
				"message": "",
				"data":    make([]interface{}, 0),
			})
			return
		}
	}

	p, err := redisqueue.NewProducerWithOptions(&redisqueue.ProducerOptions{
		StreamMaxLength:      10,
		ApproximateMaxLength: true,
	})
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "",
			"data":    make([]interface{}, 0),
		})
		return
	}

	err = p.Enqueue(&redisqueue.Message{
		Stream: "redisqueue:vercel-projects",
		Values: map[string]interface{}{
			"id": projectId,
		},
	})

	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "",
			"data":    make([]interface{}, 0),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error":   "",
		"message": "Success",
		"data":    projectId,
	})

}
