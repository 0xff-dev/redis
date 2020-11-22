package test

import "testing"

func TestZrange(t *testing.T) {
	Zrange("zset")
}

func TestHGetAll(t *testing.T) {
	HGetAll("hash")
}

//=== RUN   TestZScore
//2020/11/22 17:58:16 score:  0
//2020/11/22 17:58:16 found error redis: nil
func TestZScore(t *testing.T) {
	ZScore("zset", "member4")
	ZScore("zset", "mbm")
}