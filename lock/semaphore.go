package lock

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"log"
)

func acquireSemaphore(zsetName string, limit, timeout, now int64) (string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	identifier := uuid.New().String()
	//now := time.Now().Unix()
	log.Println("now: ", now)
	canAccess := false
	err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			items, _ := pipeliner.ZRangeByScore(context.Background(), zsetName, &redis.ZRangeBy{
				Min: "-1",
				Max: fmt.Sprintf("%d", now-timeout),
			}).Result()
			cnt, err := pipeliner.ZRemRangeByScore(context.Background(), zsetName, "-1", fmt.Sprintf("%d", now-timeout)).Result()
			if err != nil {
				log.Printf("clean %s error: %s", zsetName, err)
				return err
			}
			log.Printf("clean %d thread, items: %v\n", cnt, items)
			if _, err = pipeliner.ZAdd(context.Background(), zsetName, &redis.Z{
				Score:  float64(now),
				Member: identifier,
			}).Result(); err != nil {
				log.Printf("zadd %s for %s error: %s", identifier, zsetName, err)
				return err
			}
			log.Println("count: ", pipeliner.ZCard(context.Background(), zsetName).Val())
			// get rank
			rank := pipeliner.ZRank(context.Background(), zsetName, identifier).Val()
			if rank < limit {
				log.Printf("%s can access %s\n", identifier, zsetName)
				canAccess = true
			} else {
				pipeliner.ZRem(context.Background(), zsetName, identifier)
			}
			return nil
		})
		return err
	})
	if !canAccess {
		identifier = ""
	}
	return identifier, err
}

func releaseSemaphore(zsetName, identifier string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	cli.ZRem(context.Background(), zsetName, identifier)
}
