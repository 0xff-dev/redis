package lock

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// 如果确保在使用信号量的时候有正确的行为，最好加锁处理
func acquireFairSemaphore(semname string, limit, timeout int64) (string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	identifier := uuid.New().String() // new user uuid
	// 获取当前信号量的owner zset
	owner := semname + ":owner"
	ctr := semname + ":counter"

	now := time.Now().Unix()
	if _, err := cli.ZRemRangeByScore(context.Background(), semname, "-1", fmt.Sprintf("%d", now-timeout)).Result(); err != nil {
		log.Fatalf("delete %s's zset members error: %s", semname, err)
		return "", err
	}
	// 清理过期的数据
	if _, err := cli.ZInterStore(context.Background(), owner, &redis.ZStore{
		Keys:    []string{owner, semname},
		Weights: []float64{1.0, 0.0},
	}).Result(); err != nil {
		log.Fatalln(err)
		return "", err
	}

	// 放入相应的队列
	nowVal := cli.Incr(context.Background(), ctr).Val() // 增加counter
	cli.ZAdd(context.Background(), semname, &redis.Z{
		Score:  float64(now),
		Member: identifier,
	})
	cli.ZAdd(context.Background(), owner, &redis.Z{
		Score:  float64(nowVal),
		Member: identifier,
	})

	// 获取rank，查看当前的用户是否可以去获取改信号量
	rank := cli.ZRank(context.Background(), owner, identifier).Val()
	if rank < limit {
		// 可以获取锁
		return identifier, nil
	}

	// 不可以获取相关信号量，需要删除必要的信息
	cli.ZRem(context.Background(), semname, identifier)
	cli.ZRem(context.Background(), ctr, identifier)
	return "", fmt.Errorf("can't get semaphore, members are full")
}

func releaseFairSemaphore(semname, identifier string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}
	cli.Watch(context.Background(), func(tx *redis.Tx) error {
		tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			_, err := pipeliner.ZRem(context.Background(), semname, identifier).Result()
			if err != nil {
				return err
			}
			_, err = pipeliner.ZRem(context.Background(), semname+":owner", identifier).Result()
			return err
		})
		return nil
	})
}

func refreshFairSemaphore(semname, identifier string) bool {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	// 用户使用信号量的时间有semname提供，所以只需要更新这个即可，在做inter的时候不会删除这个使用者了。
	if _, err := cli.ZAdd(context.Background(), semname, &redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: identifier,
	}).Result(); err != nil {
		releaseFairSemaphore(semname, identifier)
		return false
	}
	return true
}
