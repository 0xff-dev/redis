package lock

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestSemaphore(t *testing.T) {
	timeout, limit := int64(15), int64(2)
	semaphore := "lock:semaphore"
	random := []int64{18, 65, 10, 9999, 1324, 5678, 134783, 7874834, 71, 0}
	//now := time.Now().Unix()
	//identifier, _ := acquireSemaphore(semaphore, limit, timeout, now)
	//t.Logf("%d acquire %s successfully, result: %s\nafter 3s, will release semaphore", 7, semaphore, identifier)
	//<- time.NewTicker().C
	//releaseSemaphore(semaphore, identifier)
	lock := make(chan struct{}, 1)
	var wg sync.WaitGroup
	for thread := 0; thread < 10; thread++ {
		r := rand.New(rand.NewSource(random[thread]))
		now := time.Now().Unix() + r.Int63()
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			lock <- struct{}{}
			identifier, err := acquireSemaphore(semaphore, limit, timeout, now)
			if err != nil {
				t.Errorf("%d acquire %s error: %s\n", key, semaphore, err)
				return
			}
			t.Logf("%d acquire %s successfully, result: %s\nafter 3s, will release semaphore", key, semaphore, identifier)
			<-time.NewTicker(5).C
			releaseSemaphore(semaphore, identifier)
			<-lock
		}(thread)

	}
	wg.Wait()
	t.Log("done")
}
