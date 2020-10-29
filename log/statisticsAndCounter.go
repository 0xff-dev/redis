package log

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var PRECISION = []int64{1, 5, 60, 100, 300, 3600, 18000, 86400}

const (
	knownKey   = "known:"
	counterKey = "count:"
)

func updateCounter(name string, count int64) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	now := time.Now().Unix()
	if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			for _, prec := range PRECISION {
				pNow := now / prec * prec
				hash := fmt.Sprintf("%d:%s", prec, name) // counter
				pipeliner.ZAdd(context.Background(), knownKey, &redis.Z{
					Score:  0,
					Member: hash,
				})
				pipeliner.HIncrBy(context.Background(), counterKey+hash, fmt.Sprintf("%d", pNow), count)
			}
			return nil
		})
		return err
	}); err != nil {
		log.Fatalf("want to update %s counter error: %s", name, err)
	}
}

type counterType struct {
	Time  int64
	Count int64
}

// return ??
func getCounter(name string, precision int64) ([]counterType, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	hash := fmt.Sprintf("%d:%s", precision, name)
	result, err := cli.HGetAll(context.Background(), counterKey+hash).Result()
	if err != nil {
		log.Fatalf("get counter information error: %s", err)
		return nil, err
	}
	values := make([]counterType, 0)
	for key, val := range result {
		_time, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			log.Fatalf("parse %s error: %s", key, err)
			return nil, err
		}
		cnt, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			log.Fatalf("parse count %s error: %s", val, err)
			return nil, err
		}
		values = append(values, counterType{
			Time:  _time,
			Count: cnt,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Time == values[j].Time {
			return values[i].Count < values[j].Count
		}
		return values[i].Time < values[j].Time
	})
	return values, nil
}
