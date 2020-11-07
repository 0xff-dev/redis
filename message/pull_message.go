/*
	chat:xxxx zset 群组信息，每个用户读取到的最大的消息id
	seen:xxxx zset 用户所属的所有群组，分值为在相应群里读取的最大的消息ID
	ids:chatID string 群里的消息id
	msgs:chatID zset 存储群组消息的集合， 值是当前的消息ID
*/
package message

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type Message struct {
	ID     int64     `json:"id"`
	Time   time.Time `json:"time"`
	Sender string    `json:"sender"`
	Msg    string    `json:"msg"`
}

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

func createChat(sender, message, chatID string, recipients []string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	if chatID == "" {
		chatID = cli.Incr(context.Background(), "chat:id").String()
	}

	recipients = append(recipients, sender)
	cli.Watch(context.Background(), func(tx *redis.Tx) error {
		_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			for _, u := range recipients {
				pipeliner.ZAdd(context.Background(), "chat:"+chatID, &redis.Z{
					Score:  0,
					Member: u,
				})
				pipeliner.ZAdd(context.Background(), "seen:"+u, &redis.Z{
					Score:  0, // max message id
					Member: chatID,
				})
			}
			return nil
		})
		return err
	})
	// todo send message
	log.Println(message)
}

func sendMessage(chatID, sender, message string) {
	// todo get lock

	if cli == nil {
		redisCli("localhost:6379", "")
	}

	msgID := cli.Incr(context.Background(), "ids:"+chatID).Val()
	now := time.Now().Unix()
	msg := Message{
		ID:     msgID,
		Time:   time.Now(),
		Sender: sender,
		Msg:    message,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Fatalln(err)
		return
	}
	cli.ZAdd(context.Background(), "msgs:"+chatID, &redis.Z{
		Score:  float64(msgID),
		Member: string(msgBytes),
	})
}

func fetchMsg(recipient string) (interface{}, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		// todo trans users and get all messages which don't read.
		chats := tx.ZRangeWithScores(context.Background(), "seen:"+recipient, 0, -1).Val()
		msgMap := map[string][]string{}
		for _, chat := range chats {
			chatID := chat.Member.(string)
			chatMsgs := tx.ZRangeByScore(context.Background(), "msgs:"+chatID, &redis.ZRangeBy{
				Min: fmt.Sprintf("%f", chat.Score+1),
				Max: fmt.Sprintf("%d", math.MaxInt64),
			}).Val()
			if _, ok := msgMap[chatID]; !ok {
				msgMap[chatID] = make([]string, 0)
			}
			msgMap[chatID] = append(msgMap[chatID], chatMsgs...)
		}

		chatUpdateMsgID := make(map[string]int64)
		for chat, msgs := range msgMap {
			if len(msgs) == 0 {
				continue
			}
			msgArr := make([]Message, 0)
			for _, msg := range msgs {
				// parse Message object
				var obj Message
				if err := json.Unmarshal([]byte(msg), &obj); err != nil {
					log.Println(err)
					continue
				}
				msgArr = append(msgArr, obj)
			}

			latestMsgId := msgArr[len(msgArr)-1].ID
			tx.ZAdd(context.Background(), "chat:"+chat, &redis.Z{
				Score:  float64(latestMsgId),
				Member: recipient,
			})

			chatUpdateMsgID[chat] = latestMsgId // update chat and seen
			midID := tx.ZRangeWithScores(context.Background(), "chat:"+chat, 0, 0).Val()
			tx.ZAdd(context.Background(), "seen:"+recipient, &redis.Z{
				Score:  float64(latestMsgId),
				Member: chat,
			})
			if len(midID) > 0 {
				tx.ZRemRangeByScore(context.Background(), "msgs:"+chat, "0", fmt.Sprintf("%d", int64(midID[0].Score)))
			}
		}
		_, _ = tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func joinChat(chatID, user string) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	idStr := cli.Get(context.Background(), "ids:"+chatID).Val()
	latestMsgId, _ := strconv.ParseFloat(idStr, 64)
	cli.ZAdd(context.Background(), "chat:"+chatID, &redis.Z{
		Score:  latestMsgId,
		Member: user,
	})
	cli.ZAdd(context.Background(), "seen:"+user, &redis.Z{
		Score:  latestMsgId,
		Member: chatID,
	})
}

func leaveChat(chatID, user string) error {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		tx.ZRem(context.Background(), "chat:"+chatID, user)
		tx.ZRem(context.Background(), "seen:"+user, chatID)
		chatMembers := tx.ZCard(context.Background(), "chat:"+chatID).Val()
		if chatMembers == 0 {
			tx.Del(context.Background(), "msgs:"+chatID)
			tx.Del(context.Background(), "ids:"+chatID)
		} else {
			midId := tx.ZRangeWithScores(context.Background(), "chat:"+chatID, 0, 0).Val()
			tx.ZRemRangeByScore(context.Background(), "msgs:"+chatID, "0", fmt.Sprintf("%d", int64(midId[0].Score)))
		}
		return nil
	})
	return err
}
