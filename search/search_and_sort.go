package search

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	_zinterset = "zinterset"
	_zunion    = "zunion"
)

func searchAndSort(query, id string, ttl, update, vote, start, num int64, desc bool) (int64, []string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	cards := int64(-1)
	reslut := make([]string, 0)
	if id != "" {
		boolR, err := cli.Expire(context.Background(), id, time.Duration(ttl)).Result()
		if err != nil {
			return cards, nil, err
		}
		if !boolR {
			id = ""
		}
	}

	if id != "" {
		id = parseAndSearch(query, int(ttl))
		scoreSearch := map[string]int64{
			id:            0,
			"sort:update": update,
			"sort:votes":  vote,
		}
		// article are sorted by zset.
		id, _ = zinterset(scoreSearch, 60)
	}

	if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		var err error
		_, err = tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			cards, err = pipeliner.ZCard(context.Background(), id).Result()
			if err != nil {
				return err
			}

			if desc {
				reslut, err = pipeliner.ZRevRange(context.Background(), id, start, start+num-1).Result()
			} else {
				reslut, err = pipeliner.ZRange(context.Background(), id, start, start+num-1).Result()
			}
			return err
		})
		return err
	}); err != nil {
		return -1, nil, err
	}

	return cards, reslut, nil
}

func zsetCommon(method string, scores map[string]int64, ttl int, args ...interface{}) (string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	id := uuid.New().String()

	items := make([]string, 0)
	if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		for k := range scores {
			items = append(items, k)
		}

		_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			var err error
			destination := idx + id
			if method == _zinterset {
				_, err = pipeliner.ZInterStore(context.Background(), destination, &redis.ZStore{
					Keys: items,
				}).Result()
			}
			if method == _zunion {
				_, err = pipeliner.ZUnionStore(context.Background(), destination, &redis.ZStore{
					Keys: items,
				}).Result()
			}
			if err != nil {
				return err
			}
			_, err = pipeliner.Expire(context.Background(), destination, time.Duration(ttl)).Result()
			return err
		})
		return err
	}); err != nil {
		return "", err
	}

	return id, nil
}

func zinterset(items map[string]int64, ttl int, args ...interface{}) (string, error) {
	return zsetCommon(_zinterset, items, ttl, args...)
}

func zunion(items map[string]int64, ttl int, args ...interface{}) (string, error) {
	return zsetCommon(_zunion, items, ttl, args...)
}
