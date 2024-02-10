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

	"github.com/go-redis/redis"
)

const (
	accessKey  = "9244ff252a3d4d8975c52c07e6a3653b"
	secretKey  = "a2c0cd7a40efc6360689357a4346be92313f004107bf3548bf283f65e8987061"
	region     = "wnam" // specify your region
	endpoint   = "https://250397b01822ad832478cabd941e8740.r2.cloudflarestorage.com"
	bucketName = "vercel-clone"
	timeout    = 10 * time.Minute
	streamName = "redisqueue:vercel-projects"
)

type Request struct {
	ProjectURL string `json:"project_url"`
}

var redisClient *redis.Client

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", "127.0.0.1", "6379"),
	})

	_, err := redisClient.Ping().Result()
	if err != nil {
		log.Fatal("Error connecting to Redis", err)
	}

	log.Println("Connected to Redis server")
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

	// p, err := redisqueue.NewProducerWithOptions(&redisqueue.ProducerOptions{
	// 	StreamMaxLength:      10,
	// 	ApproximateMaxLength: true,
	// })
	// if err != nil {
	// 	log.Println(err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{
	// 		"error":   err.Error(),
	// 		"message": "",
	// 		"data":    make([]interface{}, 0),
	// 	})
	// 	return
	// }

	// err = p.Enqueue(&redisqueue.Message{
	// 	Stream: "redisqueue:vercel-projects",
	// 	Values: map[string]interface{}{
	// 		"id": projectId,
	// 	},
	// })

	err = redisClient.XAdd(&redis.XAddArgs{
		Stream:       streamName,
		MaxLen:       0,
		MaxLenApprox: 0,
		ID:           "",
		Values: map[string]interface{}{
			"id":     projectId,
			"status": "uploaded",
		},
	}).Err()

	redisClient.HSet("status", projectId, "uploaded")

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

func Status(c *gin.Context) {
	// Extract project ID from query parameters
	projectId := c.Query("id")

	status, _ := redisClient.HGet("status", projectId).Result()

	c.JSON(http.StatusOK, gin.H{
		"id":     projectId,
		"status": status,
	})
}
