package queue

import (
	"testing"
	"time"
)

func TestQueueEmail(t *testing.T) {
	task := Task{
		SellerID: "1111",
		ItemID:   "1111",
		Price:    0,
		BuyerID:  "2222",
		Time:     time.Now(),
	}
	queueName := "queue:email"
	processEmail(queueName)
	<-time.NewTicker(20).C
	pushTashToQueue(queueName, task)
}
