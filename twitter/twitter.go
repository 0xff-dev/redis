package twitter

import (
	"context"
	"fmt"
	lock2 "github.com/stevenshuang/redis/lock"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	once sync.Once
	cli  *redis.Client
)

const (
	layout      = "2006-01-02 15:04:05"
	userPrefix  = "user:"
	usersPrefix = "users:"
	status      = "status:"
	userID      = "user:id"
	statusID    = "status:id"
	followers   = "followers:"
	following   = "following:"
	profile = "profile:"

	HOME_TIMELINE_SIZe = 1000
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

func createUser(login, name string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	lowLogin := strings.ToLower(login)
	lock := lock2.AcquireLock(userPrefix+lowLogin, 1)
	if lock == "" {
		// acquire lock failed
		return
	}

	userID, err := cli.HGet(context.Background(), usersPrefix, lowLogin).Result()
	if err == nil && userID != "" {
		// user already exists.
		lock2.ReleaseLock(userPrefix+lowLogin, lock)
		return
	}

	// new user id
	id := cli.Incr(context.Background(), userID)
	cli.Watch(context.Background(), func(tx *redis.Tx) error {
		tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			pipeliner.HSet(context.Background(), usersPrefix, lowLogin, id)
			userInfo := []interface{}{
				"login", login, "id", id, "name", name,
				"followers", 0, "following", 0, "posts", 0, "signup", time.Now().Format(layout),
			}
			pipeliner.HMSet(context.Background(), fmt.Sprintf("user:%s", id), userInfo...)
			return nil
		})
		return nil
	})
}

func createStatus(uid, message string) (string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}
	var (
		newStatusID   int64
	)
	err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		var (
			loginUserName string
			err           error
		)
		_, err = tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			loginUserName, err = pipeliner.HGet(context.Background(), userPrefix+uid, "login").Result()
			if err != nil {
				return err
			}

			newStatusID, err = tx.Incr(context.Background(), statusID).Result()
			if err != nil {
				return err
			}

			// all data is right
			statusMsg := []interface{}{
				"message", message, "posted", time.Now().Unix(), "id", newStatusID, "uid", uid, "login", loginUserName,
			}
			_, err = pipeliner.HMSet(context.Background(), status+fmt.Sprintf("%d", newStatusID), statusMsg...).Result()
			if err != nil {
				return err
			}

			_, err = pipeliner.HIncrBy(context.Background(), userPrefix+uid, "posts", 1).Result()
			return err
		})

		return err
	})
	return fmt.Sprintf("%d", newStatusID), err
}

func getStatusMessages(uid, timeline string, page, count int64) ([]map[string]string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	statuses := cli.ZRevRange(context.Background(), timeline+uid, (page-1)*count, page*count-1).Val()
	result := make([]map[string]string, 0)
	err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			for _, id := range statuses {
				r := pipeliner.HGetAll(context.Background(), status+id).Val()
				result = append(result, r)
			}
			return nil
		})
		return nil
	})
	return result, err
}

func followUser(uid, otherUid string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	// add following, and other user followers.
	fKey1 := following+uid
	fKey2 := followers+otherUid

	if _, err := cli.ZScore(context.Background(), fKey1, otherUid).Result(); err == nil {
		// uid has followed otherUid
		return
	}

	now := time.Now().Unix()
	cli.Watch(context.Background(), func(tx *redis.Tx) error {
		_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			pipeliner.ZAdd(context.Background(), fKey1, &redis.Z{
				Score:  float64(now),
				Member: otherUid,
			})
			pipeliner.ZAdd(context.Background(), fKey2, &redis.Z{
				Score:  float64(now),
				Member: uid,
			})
			latestMsgs, err := pipeliner.ZRevRangeWithScores(context.Background(), profile+otherUid, 0, HOME_TIMELINE_SIZe-1).Result()
			if err != nil {
				log.Printf("get %s latest msgs error: %s", otherUid, err)
				return err
			}
			
			pipeliner.HIncrBy(context.Background(), userPrefix+uid, "following", 1)
			pipeliner.HIncrBy(context.Background(), userPrefix+otherUid, "followers", 1)
			// why zrange don't return pointer?
			if len(latestMsgs) > 0 {
				pointerMsgs := make([]*redis.Z, 0)
				for _, z := range latestMsgs {
					pointerMsgs = append(pointerMsgs, &redis.Z{
						Score:  z.Score,
						Member: z.Member,
					})
				}
				pipeliner.ZAdd(context.Background(), "home:"+uid, pointerMsgs...)
			}
			// current home line has 1000 latest messages.
			pipeliner.ZRemRangeByRank(context.Background(), "home:"+uid, 0, -HOME_TIMELINE_SIZe)
			return nil
		})
		return err
	})
}

func unfollowUser(uid, otherUid string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}
	
	fKey1 := following+uid
	fKey2 := followers+otherUid
	
	if _, err := cli.ZScore(context.Background(), fKey1, otherUid).Result(); err != nil {
		// not follow other uid
		return
	}
	
	cli.Watch(context.Background(), func(tx *redis.Tx) error {
		tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			pipeliner.ZRem(context.Background(), fKey1, otherUid)
			pipeliner.ZRem(context.Background(), fKey2 , uid)
			latestMsgs, err := pipeliner.ZRevRange(context.Background(), profile+otherUid, 0, HOME_TIMELINE_SIZe-1).Result()
			if err != nil {
				return err
			}

			pipeliner.HIncrBy(context.Background(), userPrefix+uid, "following", -1)
			pipeliner.HIncrBy(context.Background(), userPrefix+otherUid, "followers", -1)
			if len(latestMsgs) > 0 {
				interfaceMsgs := make([]interface{}, 0)
				for _, msg := range latestMsgs {
					interfaceMsgs = append(interfaceMsgs, msg)
				}
				pipeliner.ZRem(context.Background(), "home:"+uid, interfaceMsgs...)
			}
			return nil
		})
		return nil
	})
}

func postStatus(uid, message string) string {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	// message: sss, posted: time, id:id, uid: uid, login: user
	id, err := createStatus(uid, message)
	if err != nil {
		log.Fatal(err)
	}

	timestampStr, err := cli.HGet(context.Background(), status+id, "posted").Result()
	if err != nil || timestampStr == "" {
		return ""
	}
	timestamp, _ := strconv.ParseFloat(timestampStr, 64)

	// personal timeline, with article id.
	cli.ZAdd(context.Background(), profile+uid, &redis.Z{
		Score:  timestamp,
		Member: id,
	})

	// todo success, should notice someone.
	return id
}

func deleteStatus(uid, statusID string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	key := status+statusID
	statusUid, err := cli.HGet(context.Background(), key, "uid").Result()
	if err != nil {
		log.Println(err)
		return
	}

	cli.Watch(context.Background(), func(tx *redis.Tx) error {
		tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			pipeliner.Del(context.Background(), key)
			// delete home page timeline
			pipeliner.ZRem(context.Background(), "home:"+statusUid)
			// delete profile page timeline
			pipeliner.ZRem(context.Background(), profile+statusUid)
			// reduce the number of posts.
			pipeliner.HIncrBy(context.Background(), userPrefix+statusUid, "posts", -1)
			return nil
		})
		return nil
	})
}



