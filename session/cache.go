package session

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"strconv"
	"time"
)

const (
	cacheKey    = "cache:"
	delayKey    = "delay:"
	scheduleKey = "schedule:"
)

// cache request page
func cacheRequest(request string, callback func(string) string) string {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	pageCacheKey := cacheKey + hashRequest(request)
	if r, err := cli.Get(context.Background(), pageCacheKey).Result(); err == nil {
		return r
	}

	// can't cache request page
	content := callback(request)
	cli.Set(context.Background(), pageCacheKey, content, 300)
	return content
}

func hashRequest(request string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(request)))
}

func scheduleRowCache(rowID string, delay int) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	cli.ZAdd(context.Background(), delayKey, &redis.Z{
		Score:  float64(delay),
		Member: rowID,
	})
	cli.ZAdd(context.Background(), scheduleKey, &redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: rowID,
	})
}

func cacheRows(stopChan chan struct{}) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	for {
		select {
		case <-stopChan:
			goto end
		default:
			next, err := cli.ZRange(context.Background(), scheduleKey, 0, 0).Result()
			if err != nil {
				log.Println("can't get schedule row")
				<-time.NewTimer(5 * time.Millisecond).C
				continue
			}

			floatNext, _ := strconv.ParseFloat(next[1], 10)
			now := time.Now()
			if floatNext > float64(now.Unix()) {
				log.Println("it's not time to schedule it")
				<-time.NewTimer(5 * time.Millisecond).C
				continue
			}

			// schedule rows
			rowID := next[0]
			delay := cli.ZScore(context.Background(), delayKey, rowID).Val()
			if delay <= 0 {
				log.Println("delay time is zero, delete key: ", rowID)
				// delete key
				cli.ZRem(context.Background(), delayKey, rowID)
				cli.ZRem(context.Background(), scheduleKey, rowID)
				cli.Del(context.Background(), "inv:"+rowID)
				continue
			}

			// reset time
			// todo read data from cache
			reqData := "something"
			// update delay time
			cli.ZAdd(context.Background(), scheduleKey, &redis.Z{
				Score:  float64(now.Add(time.Duration(int(delay))).Unix()),
				Member: rowID,
			})
			b, _ := json.Marshal([]byte(reqData))
			cli.Set(context.Background(), "inv:"+rowID, string(b), 300)
		}
	}
end:
	log.Println("cache row done")
}
