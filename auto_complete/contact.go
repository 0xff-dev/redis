package auto_complete

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
)

var (
	once sync.Once
	cli  *redis.Client
)

const (
	userContactsKey = "recent:"
	validCharacter  = "`abcdefghijklmnopqrstuvwxyz{"
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

func addUpdateContact(user, contact string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	acList := userContactsKey + user
	if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			// exists multi contact??
			if _, err := pipeliner.LRem(context.Background(), acList, 1, contact).Result(); err != nil {
				return err
			}
			if _, err := pipeliner.LPush(context.Background(), acList, contact).Result(); err != nil {
				return err
			}
			_, err := pipeliner.LTrim(context.Background(), acList, 0, 99).Result()
			return err
		})
		return err
	}); err != nil {
		log.Println(err)
	}
}

func removeContact(user, contact string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	cli.LRem(context.Background(), userContactsKey+user, 1, contact)
}

func fetchAutoComplete(user, prefix string) ([]string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	candidates, err := cli.LRange(context.Background(), userContactsKey+user, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for _, contact := range candidates {
		if strings.HasPrefix(contact, prefix) {
			result = append(result, contact)
		}
	}
	return result, nil
}

// 想要查找abc需要找到ab?  abc   abc?这个范围
func findPrefixRange(prefix string) (string, string) {
	position := binarySearch(prefix[len(prefix)-1])
	if position == -1 || position == 0 {
		// name can't include `
		log.Println("not validated character")
		return "", ""
	}

	pre := position - 1
	prefixStr := prefix[:len(prefix)-1]
	suffixChar := validCharacter[pre]
	return prefixStr + string(suffixChar) + "{", prefix + "{"
}

func autoCompleteByPrefix(prefix, guild string) []string {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	contacts := make([]string, 0)
	start, end := findPrefixRange(prefix)
	group := "members:" + guild
	cli.ZAdd(context.Background(), group, &redis.Z{
		Score:  0,
		Member: start,
	}, &redis.Z{
		Score:  0,
		Member: end,
	})
	if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		_, err := tx.Watch(context.Background(), group).Result()
		if err != nil {
			tx.Unwatch(context.Background())
			return err
		}
		_, err = tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			// todo maybe it's a bug, pipliner can't get zset list.
			startIndex, err := pipeliner.ZRank(context.Background(), group, start).Result()
			if err != nil {
				log.Println("get start err: ", err)
				return err
			}
			endIndex, err := pipeliner.ZRank(context.Background(), group, end).Result()
			if err != nil {
				log.Println("get end err: ", err)
				return err
			}
			tmpContacts, err := pipeliner.ZRange(context.Background(), group, startIndex, endIndex).Result()
			if err != nil {
				return err
			}
			for _, item := range tmpContacts {
				if !strings.Contains(item, "{") {
					contacts = append(contacts, item)
				}
			}
			return nil
		})
		return err
	}); err != nil {
		return nil
	}

	return contacts
}

func joinOrLeaveGuild(guild, user string, join bool) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}
	key := "members:" + guild
	if join {
		cli.ZAdd(context.Background(), key, &redis.Z{
			Score:  0,
			Member: user,
		})
		log.Println("add: ", cli.ZRange(context.Background(), key, 0, -1).Val())
		return
	}
	cli.ZRem(context.Background(), key, user)
	log.Println("delete: ", cli.ZRange(context.Background(), key, 0, -1).Val())
}

func binarySearch(prefix byte) int {
	bytes := []byte(validCharacter)
	start, end := 0, len(bytes)-1
	for start <= end {
		mid := start + (end-start)/2
		if bytes[mid] == prefix {
			return mid
		} else if bytes[mid] < prefix {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	return -1
}
