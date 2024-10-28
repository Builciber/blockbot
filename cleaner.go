package main

import (
	"context"
	"log"
	"sync"
	"time"
)

func cleaner(ctx context.Context, interval time.Duration, intSeqMap map[chatID]*interactionSequence, mu *sync.RWMutex) {
	log.Println("Starting interaction sequence cleaner")
	ticker := time.NewTicker(interval)
	tickerChan := ticker.C
	for {
		select {
		case <-tickerChan:
			currentTime := time.Now()
			mu.Lock()
			for k, v := range intSeqMap {
				if currentTime.Sub(v.createdAt) >= 1*time.Hour {
					delete(intSeqMap, k)
				}
			}
			mu.Unlock()
		case <-ctx.Done():
			log.Println("Stopping interaction sequence cleaner: Context done")
			ticker.Stop()
			return
		}
	}
}
