package command

import "testing"

func TestPubAndSubscribe(t *testing.T) {
	channel := "chan"
	go subscribeMessage(channel)
	publish(channel)
}
