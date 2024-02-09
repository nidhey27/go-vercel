package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/robinjoseph08/redisqueue"
)

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
	log.Printf("processing message: %v\n", msg.Values["id"])
	projectID := msg.Values["id"]

	if _, err := os.Stat("./output" + projectID.(string)); err == nil {
		err := DownloadS3Folder(projectID.(string))

		if err != nil {
			log.Println(err)
			return err
		}
	}

	err := buildProject(fmt.Sprintf("./output/%v", projectID))

	if err != nil {
		log.Println(err)
		return err
	}

	err = UploadFilesToS3(projectID.(string))

	if err != nil {
		log.Println(err)
		return err
	}

	p, err := redisqueue.NewProducerWithOptions(&redisqueue.ProducerOptions{
		StreamMaxLength:      10,
		ApproximateMaxLength: true,
	})

	if err != nil {
		return err
	}

	err = p.Enqueue(&redisqueue.Message{
		Stream: "redisqueue:vercel-projects",
		Values: map[string]interface{}{
			"id":     projectID,
			"status": "deployed",
		},
	})

	if err != nil {
		return err
	}

	return nil
}
