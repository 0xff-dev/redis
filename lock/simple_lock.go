package lock

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)



var (
	once sync.Once
	cli *redis.Client
)

const (
	lockKey = "lock:"
)


func redisCli(host, password string) {
	once.Do(func() {
		cli = redis.NewClient(&redis.Options{
			Addr: host,
			Password: password,
			DB: 0,
		})
	})
}

func AcquireLock(lockName string, timeout int) string {
	if cli == nil {
		redisCli("localhost:6379", "")
	}
	var identifier interface{}
	identifier = uuid.New().String()
	end := time.Now().Unix()+int64(timeout)

	for time.Now().Unix() < end {
		if _, err := cli.SetNX(context.Background(), lockKey+lockName, identifier, 0).Result(); err == nil {
			return identifier.(string)
		}
		time.Sleep(time.Millisecond)
	}
	return ""
}

func ReleaseLock(locakName, identifier string) bool {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	lock := lockKey+locakName
	for {
		err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
			_, err := tx.Watch(context.Background(), lock).Result()
			if err != nil {
				return err
			}

			if cli.Get(context.Background(), lock).Val() == identifier {
				tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
					_, err := pipeliner.Del(context.Background(), lock).Result()
					return err
				})
			}
			tx.Unwatch(context.Background())
			return nil
		})
		if err == nil {
			return true
		}
		break
	}
	return false
}


