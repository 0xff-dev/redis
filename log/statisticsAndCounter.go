package log

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stevenshuang/redis/utils"
)

var PRECISION = []int64{1, 5, 60, 100, 300, 3600, 18000, 86400}

const (
	knownKey   = "known:"
	counterKey = "count:"

	SAMPLE_COUNT = 10
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

func cleanCounters() {
	if cli == nil {
		redisCli("localhost:6379", "")
	}
	passes := int64(0)
	for {
		start := time.Now().Unix()
		index := int64(0)
		knownSets := cli.ZCard(context.Background(), knownKey).Val()
		for index < knownSets {
			hashItems := cli.ZRange(context.Background(), knownKey, index, index).Val()
			index++
			if len(hashItems) == 0 {
				break
			}
			// 5:hits
			hash := hashItems[0]
			splitHash := strings.Split(hash, ":")
			interval, err := strconv.ParseInt(splitHash[0], 10, 64)
			if err != nil {
				break
			}
			bInterval := int64(1)
			if interval%60 != 0 {
				bInterval = interval / 60
			}

			if passes%interval != 0 {
				continue
			}
			hkey := counterKey + hash
			cutoff := fmt.Sprintf("%d", time.Now().Unix()-SAMPLE_COUNT*bInterval)
			hkeyItems := cli.HKeys(context.Background(), hkey).Val()
			sort.Slice(hkeyItems, func(i, j int) bool {
				return compareStrByInt(hkeyItems[i], hkeyItems[j])
			})

			foundIndex := binarySearch(hkeyItems, cutoff)
			if foundIndex != -1 {
				cli.HDel(context.Background(), hkey, hkeyItems[:foundIndex+1]...)
				if foundIndex == len(hkeyItems)-1 {
					// remove all, try to delete key
					if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
						tx.Watch(context.Background(), hkey)
						if tx.HLen(context.Background(), hkey).Val() != 0 {
							tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
								_, err := pipeliner.ZRem(context.Background(), knownKey, hash).Result()
								return err
							})
							index--
						} else {
							tx.Unwatch(context.Background())
						}
						// we should watch error
						return nil
					}); err != nil {
						log.Println(err)
					}
				}
			}
		}
		passes++
		duration := utils.Int64Min(time.Now().Unix()-start+1, 60)
		<-time.NewTicker(time.Duration(utils.Int64Max(60-duration, 1))).C
	}
}

func binarySearch(items []string, item string) int {
	start, end := 0, len(items)-1
	for start <= end {
		mid := start + (end-start)/2
		if items[mid] == item {
			return mid
		} else if compareStrByInt(items[mid], item) {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	return -1
}

func compareStrByInt(a, b string) bool {
	i64, _ := strconv.ParseInt(a, 10, 64)
	j64, _ := strconv.ParseInt(a, 10, 64)
	return i64 < j64
}
