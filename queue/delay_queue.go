// 延迟队列的思想就是，当任务的延时为0的时候直接推向发送队列，都是记录到zset中，时间戳做分值。
// 一个协程轮询
package queue

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"log"
	"time"
)

func executeLater(queueName string, task Task, delay int64) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	identifier := uuid.New().String()
	task.Description = identifier
	taskBytes, err := json.Marshal(task)
	if err != nil {
		log.Fatal(err)
		return
	}
	// 有延时，放到延时队列，否则知道放到执行队列即可。
	if delay > 0 {
		cli.ZAdd(context.Background(), "delayed:", &redis.Z{
			Score:  float64(time.Now().Unix() + delay),
			Member: string(taskBytes),
		})
		return
	}
	cli.RPush(context.Background(), "queue:"+queueName, string(taskBytes))
}

func pollQueue() {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	for {
		items := cli.ZRange(context.Background(), "delayed:", 0, 0).Val()
		if len(items) == 0 {
			log.Println("empty delayed queue")
			time.Sleep(time.Millisecond)
			continue
		}
		// has some value
		taskString := items[0]
		var task Task
		if err := json.Unmarshal([]byte(taskString), &task); err != nil {
			log.Println(err)
			continue
		}
		// todo execute task.
		cli.RPush(context.Background(), "queue:"+task.QueueName, taskString)
	}
}
