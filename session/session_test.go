package session

import (
	"crypto/md5"
	"fmt"
	"log"
	"testing"
	"time"
)

func randomStr() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
}

func TestSession(t *testing.T) {
	stopChan := make(chan struct{})

	for idx := 0; idx < 15; idx++ {
		token := randomStr()
		log.Println("random token: ", token)
		updateToken(token, fmt.Sprintf("%d", idx), "")

		log.Println("check token: ", checkToken(token))
		time.Sleep(2 * time.Second)
	}
	sessionInfo()
	go func() {
		cleanSessions(stopChan, 8)
	}()
	t.Log("after 30s, stop clean sessions")
	<-time.NewTimer(30 * time.Second).C
	stopChan <- struct{}{}

	sessionInfo()
}
