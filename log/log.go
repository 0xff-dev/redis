package log

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	once sync.Once
	cli  *redis.Client
)

const timeFormat = "2006 01/02 15:04:05"

func redisCli(host, password string) {
	once.Do(func() {
		cli = redis.NewClient(&redis.Options{
			Addr:     host,
			Password: password,
			DB:       0,
		})
	})
}

func logRecent(name, msg, level string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	destination := fmt.Sprintf("recent:%s:%s", name, level)
	message := time.Now().Format(timeFormat) + " " + msg
	if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			pipeliner.LPush(context.Background(), destination, message)
			pipeliner.LTrim(context.Background(), destination, 0, 99)
			return nil
		})
		return nil
	}); err != nil {
		log.Fatalf("tx watch error: %s", err)
	}
}
