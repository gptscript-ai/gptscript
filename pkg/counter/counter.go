package counter

import (
	"fmt"
	"sync/atomic"
	"time"
)

var counter = int32(time.Now().Unix())

func Reset(i int32) {
	atomic.StoreInt32(&counter, i)
}

func Next() string {
	return fmt.Sprint(atomic.AddInt32(&counter, 1))
}
