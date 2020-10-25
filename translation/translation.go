/* 用户信息(hash)+包裹信息(hash)+市场信息(zset), 事务处理 */
package translation

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	userKey   = "users:"
	inventory = "inventory:"
	market    = "market:"
)

var (
	once sync.Once
	cli  *redis.Client
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

func listItems(itemID, sellerID string, price int) bool {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	userInventory := fmt.Sprintf("%s%s", inventory, sellerID)
	item := fmt.Sprintf("%s:%s", itemID, sellerID)
	end := time.Now().Unix() + 5

	for time.Now().Unix() < end {
		if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
			// watch user inventory
			if _, err := tx.Watch(context.Background(), userInventory).Result(); err != nil {
				return err
			}
			if !tx.SIsMember(context.Background(), userInventory, itemID).Val() {
				log.Printf("%s is not member of %s\n", item, userInventory)
				tx.Unwatch(context.Background())
				return fmt.Errorf("%s is not member of %s", item, userInventory)
			}
			_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
				pipeliner.ZAdd(context.Background(), market, &redis.Z{
					Score:  float64(price),
					Member: item,
				})
				pipeliner.SRem(context.Background(), userInventory, itemID)
				return nil
			})
			return err
		}); err != nil {
			log.Println("watch error ", err)
			continue
		}
		return true
	}
	return false
}

func buyItem(buyerID, itemID, sellerID string, price int) bool {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	buyer := fmt.Sprintf("%s%s", userKey, buyerID)
	seller := fmt.Sprintf("%s%s", userKey, sellerID)
	item := fmt.Sprintf("%s:%s", itemID, sellerID)
	buyerInventory := fmt.Sprintf("%s%s", inventory, buyerID)
	end := time.Now().Unix() + 10
	for time.Now().Unix() < end {
		if err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
			if _, err := tx.Watch(context.Background(), market, buyer).Result(); err != nil {
				return err
			}

			itemPrice := tx.ZScore(context.Background(), market, item).Val()
			buyerFunds, _ := strconv.ParseFloat(tx.HGet(context.Background(), buyer, "funds").Val(), 64)
			if itemPrice != float64(price) || itemPrice > buyerFunds {
				// change price or buyer don't have enough money
				tx.Unwatch(context.Background())
				return fmt.Errorf("price error")
			}
			_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
				pipeliner.HIncrByFloat(context.Background(), seller, "funds", itemPrice)
				pipeliner.HIncrByFloat(context.Background(), buyer, "funds", -itemPrice)
				pipeliner.SAdd(context.Background(), buyerInventory, itemID)
				pipeliner.ZRem(context.Background(), market, item)
				return nil
			})
			return err
		}); err != nil {
			log.Println("watch error: ", err)
			continue
		}
		return true
	}
	return false
}

func addInfo() {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	user1 := map[string]interface{}{
		"name":  "coco",
		"funds": 33.4,
	}
	user2 := map[string]interface{}{
		"name":  "none",
		"funds": 50.0,
	}
	cli.HSet(context.Background(), fmt.Sprintf("%s%s", userKey, "1"), user1)
	cli.HSet(context.Background(), fmt.Sprintf("%s%s", userKey, "2"), user2)

	u1Inventory := []interface{}{"item1", "item2"}
	u2Inventory := []interface{}{"1111", "2222"}
	cli.SAdd(context.Background(), fmt.Sprintf("%s%s", inventory, "1"), u1Inventory...)
	cli.SAdd(context.Background(), fmt.Sprintf("%s%s", inventory, "2"), u2Inventory...)
}

func checkInfo() {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	fmt.Println("check user funds")
	u1 := cli.HGet(context.Background(), fmt.Sprintf("%s%s", userKey, "1"), "funds").Val()
	u2 := cli.HGet(context.Background(), fmt.Sprintf("%s%s", userKey, "2"), "funds").Val()

	fmt.Println("u1: ", u1, "\nu2: ", u2)

	fmt.Println("check user's inventory")
	u1Inventory := cli.SMembers(context.Background(), fmt.Sprintf("%s%s", inventory, "1")).Val()
	u2Inventory := cli.SMembers(context.Background(), fmt.Sprintf("%s%s", inventory, "2")).Val()
	fmt.Println("u1 inventory: ", u1Inventory, "\nu2 inventory: ", u2Inventory)

	fmt.Println("check market")
	markets := cli.ZRange(context.Background(), market, 0, -1).Val()
	fmt.Println("markets: ", markets)
}
