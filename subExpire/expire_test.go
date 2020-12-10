package subExpire

import "testing"

func TestExpireAndSub(t *testing.T) {
	go subExpireKey()
	addExpireItem()
}
