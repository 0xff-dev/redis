package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"sync"
	"time"
)

var (
	once sync.Once
	cli  *redis.Client
)

type Task struct {
	SellerID    string    `json:"seller_id"`
	ItemID      string    `json:"item_id"`
	Price       float64   `json:"price"`
	BuyerID     string    `json:"buyer_id"`
	Time        time.Time `json:"time"`
	Description string    `json:"description"`
	QueueName   string    `json:"queue_name"`
}

func (t Task) String() string {
	return fmt.Sprintf("%s from %s buy %s at %s, price is: %f", t.BuyerID, t.SellerID, t.ItemID, t.Time, t.Price)
}

// how to deal task
func taskWorker(t Task) error {
	log.Println(t)
	return nil
}

func redisCli(host, password string) {
	once.Do(func() {
		cli = redis.NewClient(&redis.Options{
			Addr:     host,
			Password: password,
			DB:       0,
		})
	})
}

func pushTashToQueue(queueName string, task Task) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	taskBytes, err := json.Marshal(task)
	if err != nil {
		log.Fatalf("marshal task[%s] error: %s", task, err)
		return
	}

	cli.RPush(context.Background(), queueName, string(taskBytes))
}

func processEmail(queueName string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	for {
		taskString, err := cli.BLPop(context.Background(), 10, queueName).Result()
		if err != nil || len(taskString) == 0 {
			log.Println("empty queue")
			continue
		}
		var task Task
		if err = json.Unmarshal([]byte(taskString[0]), &task); err != nil {
			log.Println(err)
			continue
		}
		log.Println("send info")
		_ = taskWorker(task)
	}
}

func workerWatchQueue(queueName string, callback func(Task)) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	for {
		taskBytes := cli.BLPop(context.Background(), 30, queueName).Val()
		if len(taskBytes) == 0 {
			continue
		}
		var task Task
		if err := json.Unmarshal([]byte(taskBytes[0]), &task); err != nil {
			log.Println(err)
			continue
		}
		// maybe there are many callback functions
		callback(task)
	}
}
