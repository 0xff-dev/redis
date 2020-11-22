// test package test some go-redis function's return value
package test

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
	"sync"
)

var (
	once sync.Once
	cli  *redis.Client
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

func Zrange(key string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	v := cli.ZRangeByScoreWithScores(context.Background(), key, &redis.ZRangeBy{
		Min:    "0",
		Max:    "99999",
	}).Val()
	log.Println(v)
}

func HGetAll(key string) {
	if cli == nil {
		redisCli("localhost:6379","")
	}

	vs := cli.HGetAll(context.Background(), key).Val()
	log.Println(vs)
}

func ZScore(key, field string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	score, err := cli.ZScore(context.Background(), key, field).Result()
	if err != nil {
		log.Fatal("found error ",err)
	}

	log.Println("score: ", score)
}
