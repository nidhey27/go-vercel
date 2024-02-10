package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	accessKey  = "9244ff252a3d4d8975c52c07e6a3653b"
	secretKey  = "a2c0cd7a40efc6360689357a4346be92313f004107bf3548bf283f65e8987061"
	region     = "wnam" // specify your region
	endpoint   = "https://250397b01822ad832478cabd941e8740.r2.cloudflarestorage.com"
	bucketName = "vercel-clone"
	timeout    = 10 * time.Minute
)

func main() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == "https://github.com"
		},
		MaxAge: 12 * time.Hour,
	}))

	router.GET("/*path", func(c *gin.Context) {

		projectID := strings.Split(c.Request.Host, ".")[0]
		requestPath := c.Request.URL.Path

		if requestPath == "/" {
			requestPath = "index.html"
		}

		config := &aws.Config{
			Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
			Endpoint:         aws.String(endpoint),
			Region:           aws.String(region),
			S3ForcePathStyle: aws.Bool(true),
		}

		sess := session.Must(session.NewSession(config))
		svc := s3.New(sess)
		// c.Header("Content-Type", "application/json; charset=utf-8")
		// c.JSON(http.StatusOK, gin.H{
		// 	"url": ,
		// })

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

		object, err := svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fmt.Sprintf("%v/dist/%v", projectID, requestPath)),
		})

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error":       err,
				"requestPath": requestPath,
			})
			return
		}

		content, err := io.ReadAll(object.Body)
		if err != nil {
			log.Println(err)
			return
		}

		contentType := ""

		if strings.Contains(requestPath, ".html") {
			contentType = "text/html"
		} else if strings.Contains(requestPath, ".css") {
			contentType = "text/css"
		} else {
			contentType = "application/javascript"
		}

		c.Data(http.StatusOK, contentType, []byte(content))
	})

	err := router.Run(":3001")
	if err != nil {
		log.Println(err)
	}
}
