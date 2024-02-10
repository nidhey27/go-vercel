package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
	"github.com/robinjoseph08/redisqueue"
)

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

func main() {
	c, err := redisqueue.NewConsumerWithOptions(&redisqueue.ConsumerOptions{
		VisibilityTimeout: 60 * time.Second,
		BlockingTimeout:   5 * time.Second,
		ReclaimInterval:   1 * time.Second,
		BufferSize:        100,
		Concurrency:       10,
	})
	if err != nil {
		panic(err)
	}

	c.Register("redisqueue:vercel-projects", process)

	go func() {
		for err := range c.Errors {
			// handle errors accordingly
			log.Printf("err: %+v\n", err)
		}
	}()

	c.Run()
}

func process(msg *redisqueue.Message) error {
	messageID := msg.ID
	projectID := msg.Values["id"]
	status := msg.Values["status"]
	if status != "deployed" {
		log.Printf("processing message: %v\n", msg.Values["id"])

		err := DownloadS3Folder(projectID.(string))

		if err != nil {
			log.Println(err)
			return err
		}

		err = buildProject(fmt.Sprintf("./output/%v", projectID))

		if err != nil {
			log.Println(err)
			return err
		}

		err = UploadFilesToS3(projectID.(string), messageID)

		if err != nil {
			log.Println(err)
			return err
		}
		redisClient.HSet("status", projectID.(string), "deployed").Err()

		if err != nil {
			return err
		}
	}

	return nil
}
