package session

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

const (
	tokenKey  = "login:"
	recentKey = "recent:"
	viewedKey = "viewed:"
	cartKey   = "cart:"

	cleanLimit = 2
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

func min(a, b int64) int64 {
	if a > b {
		return b
	}
	return a
}

func checkToken(token string) string {
	if cli == nil {
		redisCli("localhost:6379", "")
	}
	return cli.HGet(context.Background(), tokenKey, token).Val()
}

func updateToken(token, user, goods string) {
	timestamp := time.Now().Unix()
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	cli.HSet(context.Background(), tokenKey, token, user) // set token user
	cli.ZAdd(context.Background(), recentKey, &redis.Z{
		Score:  float64(timestamp),
		Member: token,
	})

	if len(goods) > 0 {
		// add recently viewed goods
		cli.ZAdd(context.Background(), viewedKey+token, &redis.Z{
			Score:  float64(timestamp),
			Member: goods,
		})
		cli.ZRemRangeByRank(context.Background(), viewedKey+token, 0, -26)
	}
}

func cleanSessions(signal chan struct{}, limit int64) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	for {
		select {
		case <-signal:
			goto end
		default:
			// clean
			size := cli.ZCard(context.Background(), recentKey).Val()
			if size > limit {
				log.Printf("size: %d > limit: %d, clean recent tokens...\n", size, limit)
				endIndex := min(size-limit, cleanLimit)
				tokens := cli.ZRange(context.Background(), recentKey, 0, endIndex-1).Val()
				sessionKeys := make([]string, 0)
				interfaceTokens := make([]interface{}, 0)
				for _, token := range tokens {
					sessionKeys = append(sessionKeys, viewedKey+token, cartKey+token) // add shopping cart session
					interfaceTokens = append(interfaceTokens, token)
				}

				log.Println("delete tokens: ", tokens)
				cli.Del(context.Background(), sessionKeys...)
				cli.HDel(context.Background(), tokenKey, tokens...)
				cli.ZRem(context.Background(), recentKey, interfaceTokens...)
			} else {
				log.Println("nothing!!!")
			}
		}
		<-time.NewTimer(time.Second * 3).C
	}
end:
	log.Println("clean up completed")
}

func sessionInfo() {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	log.Println(tokenKey, " --> ", cli.HGetAll(context.Background(), tokenKey).Val())
	log.Println(recentKey, " --> ", cli.ZRange(context.Background(), recentKey, 0, -1).Val())
}

// shopping cart
func addToCart(token, item string, count int) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	if count <= 0 {
		// remove
		cli.HDel(context.Background(), cartKey+token, item)
		return
	}
	// add
	cli.HSet(context.Background(), cartKey+token, item, count)
}
