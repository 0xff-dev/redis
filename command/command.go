package command

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	cli  *redis.Client
	once sync.Once
)

func redisCli(host, password string) {
	once.Do(func() {
		cli = redis.NewClient(&redis.Options{
			Addr:     host,
			Password: password,
			DB:       0,
		})
	})
}

func subscribeMessage(channel string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	subObj := cli.Subscribe(context.Background(), channel)
	_, err := subObj.Receive(context.Background())
	if err != nil {
		log.Fatalf("subscribe channel %s error: %s", channel, err)
		return
	}
	ch := subObj.Channel()
	for {
		select {
		case msg := <-ch:
			log.Printf("channel name: %s\n, channel pattern: %s\n, channel payload: %s\n, payloads: %v\n", msg.Channel, msg.Pattern, msg.Payload, msg.PayloadSlice)
		case <-time.NewTimer(30 * time.Second).C:
			log.Println("after 30s, will unsubscribe!!!")
			if err = subObj.Unsubscribe(context.Background(), channel); err != nil {
				log.Fatalf("unsubscribe channel %s error: %s\n", channel, err)
			}
			goto end
		default:
			log.Println("wait message")
		}
		time.Sleep(2)
	}
end:
	log.Println("done")
}

func publish(channel string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	for index := 1; index < 11; index++ {
		cli.Publish(context.Background(), channel, index)
		time.Sleep(3)
	}
}
