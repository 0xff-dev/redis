package subExpire

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	once sync.Once
	cli *redis.Client
)

const subKey = "__keyevent@0__:expired"

func redisCli(host, password string) {
	once.Do(func() {
		cli = redis.NewClient(&redis.Options{
			Addr: host,
			Password: password,
			DB: 0,
		})
	})
}

func init() {
	if cli == nil {
		redisCli("localhost:6379", "")
	}
	cli.ConfigSet(context.Background(), "notify-keyspace-events", "Ex")
}

func subExpireKey() {
	fmt.Println("sub: ", subKey)
	sub := cli.Subscribe(context.Background(), subKey)
	for {
		item, err := sub.ReceiveMessage(context.Background())
		if err != nil {
			fmt.Println("get msg error: ", err)
			continue
		}
		fmt.Printf("channel: %s, pattern: %s, payload: %s, payloadSlice: %v", item.Channel, item.Pattern, item.Payload, item.PayloadSlice)
	}
}

func addExpireItem() {

	for key := 'A'; key <= 'Z'; key++ {
		now := time.Now().Add(5*time.Second)
		fmt.Println("add key: ", string(key), " expire at: ", now.String())
		cli.Set(context.Background(), string(key), key, 0)
		cli.ExpireAt(context.Background(), string(key), now)
		time.Sleep(1*time.Second)
	}
}