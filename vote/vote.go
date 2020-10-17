package vote

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	cli  *redis.Client
	once sync.Once
)

const (
	ONE_WEEK   = 7 * 86400
	VOTE_SCORE = 432

	timePrefix    = "time:"
	articlePrefix = "article:"
	votePrefix    = "vote:"
	scorePrefix   = "score:"
	groupPrefix   = "group:"

	ARTICLES_PER_PAGE = 10
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

// articleVote vote for article
func voteArticle(user, article string) error {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	offset := time.Now().Unix() - ONE_WEEK
	if r := cli.ZScore(context.Background(), timePrefix, article); r.Val() < float64(offset) {
		return fmt.Errorf("voting time has passed")
	}

	articleID := strings.Split(article, ":")[1]
	if _, err := cli.SAdd(context.Background(), votePrefix+articleID, user).Result(); err == nil {
		cli.ZIncrBy(context.Background(), scorePrefix, float64(VOTE_SCORE), article) // zset add score
		cli.HIncrBy(context.Background(), article, "votes", 1)                       // add hash field
		return nil
	}

	return fmt.Errorf("%s vote for article %s error", user, articleID)
}

func postArticle(user, title, link string) (string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	articleID := fmt.Sprintf("%d", cli.Incr(context.Background(), articlePrefix).Val())
	voteKey := votePrefix + articleID
	articleKey := articlePrefix + articleID
	now := time.Now().Unix()

	// add article

	cli.ZAdd(context.Background(), voteKey, &redis.Z{Member: user, Score: float64(now)})
	cli.Expire(context.Background(), voteKey, ONE_WEEK)

	cli.HSet(context.Background(), articleKey, map[string]interface{}{
		"title":  title,
		"link":   link,
		"poster": user,
		"time":   fmt.Sprintf("%d", now),
		"votes":  "0",
	})
	cli.ZAdd(context.Background(), scorePrefix, &redis.Z{
		Score:  float64(now + int64(VOTE_SCORE)),
		Member: articleKey,
	})
	cli.ZAdd(context.Background(), timePrefix, &redis.Z{
		Score:  float64(now),
		Member: articleKey,
	})

	return articleID, nil
}

func getArticle(page int, order string) []map[string]string {
	start := (page - 1) * ARTICLES_PER_PAGE
	end := start + ARTICLES_PER_PAGE - 1

	if cli == nil {
		redisCli("localhost:6379", "")
	}

	articleScores := cli.ZRevRange(context.Background(), order, int64(start), int64(end))
	articles := make([]map[string]string, 0)
	for _, s := range articleScores.Val() {
		// 得分最高的文章
		articleDetail := cli.HGetAll(context.Background(), s).Val()
		articleDetail["id"] = s
		articles = append(articles, articleDetail)
	}
	return articles
}

func addOrRemoveGroups(article_id string, addGroup, remGroup []string) {
	article := articlePrefix + article_id

	if cli == nil {
		redisCli("localhost:6379", "")
	}

	for _, g := range addGroup {
		cli.SAdd(context.Background(), groupPrefix+g, article)
	}
	for _, g := range remGroup {
		cli.SRem(context.Background(), groupPrefix+g, article)
	}
}

func getGroupArticles(page int, group, order string) []map[string]string {
	groupKey := groupPrefix + group
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	if r := cli.Exists(context.Background(), groupKey); r.Val() != 1 {
		cli.ZInterStore(context.Background(), order+group, &redis.ZStore{
			Keys:      []string{groupPrefix + group, order},
			Aggregate: "max",
		})
		cli.Expire(context.Background(), order+group, 60)
	}
	return getArticle(page, order)
}
