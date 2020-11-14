package search

import (
	"bufio"
	"context"
	"github.com/google/uuid"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	once sync.Once
	cli  *redis.Client

	wordRe  = regexp.MustCompile(`[a-z']{2,}`)
	queryRe = regexp.MustCompile(`[+-]?[a-z']{2,}`)
)

const (
	idx = "idx:"

	sinterstore = "sinterstore"
	sunionstore = "sunionstore"
	sdiffstore  = "sdiffstore"
)

type set map[string]struct{}

func (s set) Add(key string) {
	s[key] = struct{}{}
}

func (s set) Del(key string) {
	delete(s, key)
}

func (s set) diff(other set) set {
	r := set{}
	for k := range s {
		if _, ok := other[k]; ok {
			continue
		}
		r[k] = struct{}{}
	}
	return r
}

func (s set) toArray() []string {
	arr := make([]string, 0)
	for k := range s {
		arr = append(arr, k)
	}

	return arr
}

func redisCli(host, password string) {
	once.Do(func() {
		cli = redis.NewClient(&redis.Options{
			Addr:     host,
			Password: password,
			DB:       0,
		})
	})
}

func readStopWords(filePath string) (set, error) {
	s := set{}
	f, err := os.Open(filePath)
	if err != nil {
		return s, err
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, w := range strings.Split(line, " ") {
			s[w] = struct{}{}
		}
	}
	return s, nil
}

func tokenize(content string) (set, error) {
	s, err := readStopWords("./stop_words")
	if err != nil {
		return set{}, err
	}
	word := set{}
	res := wordRe.FindAllString(content, -1)
	for _, str := range res {
		_s := strings.Trim(str, "'")
		if len(_s) >= 2 {
			word[_s] = struct{}{}
		}
	}
	return word.diff(s), nil
}

func indexDoc(docID, content string) error {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	s, err := tokenize(content)
	if err != nil {
		return err
	}
	for key := range s {
		cli.SAdd(context.Background(), idx+key, docID)
	}
	return nil
}

func setCommon(method string, names []string, ttl int) (string, error) {
	if cli == nil {
		redisCli("localhost:6379", "")
	}

	id := uuid.New().String()
	err := cli.Watch(context.Background(), func(tx *redis.Tx) error {
		withPrefixNames := make([]string, 0)
		for _, n := range names {
			withPrefixNames = append(withPrefixNames, idx+n)
		}

		_, err := tx.Pipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
			var err error
			destination := idx + id
			if method == sinterstore {
				_, err = pipeliner.SInterStore(context.Background(), destination, withPrefixNames...).Result()
			}
			if method == sunionstore {
				_, err = pipeliner.SUnionStore(context.Background(), destination, withPrefixNames...).Result()
			}
			if method == sdiffstore {
				_, err = pipeliner.SDiffStore(context.Background(), destination, withPrefixNames...).Result()
			}
			if err != nil {
				log.Println("set operation error: ", err)
				return err
			}
			_, err = pipeliner.Expire(context.Background(), destination, time.Duration(ttl)*time.Second).Result()
			return err
		})
		return err
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

func interSet(items []string, ttl int) (string, error) {
	return setCommon(sinterstore, items, ttl)
}

func union(items []string, ttl int) (string, error) {
	return setCommon(sunionstore, items, ttl)
}

func difference(items []string, ttl int) (string, error) {
	return setCommon(sdiffstore, items, ttl)
}

func parse(query string) ([][]string, []string) {
	stopWords, _ := readStopWords("./stop_words")
	log.Println("stop words: ", stopWords)

	unwanted, current := set{}, set{}
	all := make([][]string, 0)
	for _, q := range queryRe.FindAllString(query, -1) {
		str := q
		prefix := str[0]
		if prefix == '+' || prefix == '-' {
			str = str[1:]
		} else {
			prefix = ' '
		}

		str = strings.Trim(str, "'")
		if _, ok := stopWords[str]; ok || len(str) < 2 {
			log.Println("ignore word ", str)
			// str in stop words
			continue
		}

		if prefix == '-' {
			unwanted.Add(str)
			continue
		}

		if len(current) > 0 && prefix == ' ' {
			all = append(all, current.toArray())
			current = set{}
		}

		current.Add(str)
	}
	if len(current) > 0 {
		all = append(all, current.toArray())
	}
	return all, unwanted.toArray()
}

func parseAndSearch(query string, ttl int) string {
	all, unwanted := parse(query)

	if len(all) == 0 {
		return ""
	}

	toInterSet := make([]string, 0)
	for _, s := range all {
		if len(s) > 1 {
			// idx:uuid --> a, c, d
			items, _ := union(s, ttl)
			toInterSet = append(toInterSet, items)
		} else {
			toInterSet = append(toInterSet, s[0])
		}
	}

	var interID string
	if len(toInterSet) > 1 {
		interID, _ = interSet(toInterSet, ttl)
	} else {
		interID = toInterSet[0]
	}

	if len(unwanted) > 0 {
		unwanted = append(unwanted, interID)
		r, _ := difference(unwanted, ttl)
		return r
	}
	return interID
}
