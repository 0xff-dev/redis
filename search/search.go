package search

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
)

var (
	once sync.Once
	cli  *redis.Client

	wordRe = regexp.MustCompile(`[a-z']{2,}`)
)

const (
	idx = "idx:"
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
	for k := range other {
		if _, ok := s[k]; !ok {
			r[k] = struct{}{}
		}
	}
	return r
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
		cli.SAdd(context.Background(), idx+docID, key)
	}
	return nil
}
