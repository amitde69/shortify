package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func signalHanlder(done chan bool, sigs chan os.Signal) {
	sig := <-sigs
	fmt.Println()
	fmt.Println("sig", sig)
	done <- true
}

func main() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go signalHanlder(done, sigs)

	wg := sync.WaitGroup{}
	var count int

	go func() {
		for {

			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					start := time.Now()
					res, err := http.Get("http://localhost:5001/c3c1b1e")
					end := time.Since(start).Seconds()
					if err != nil {
						fmt.Println(err)
						return
					}
					count++
					fmt.Printf("[%d] %d %.4fs\n", count, res.StatusCode, end)
				}()
			}
			wg.Wait()
		}
	}()

	<-done
	wg.Wait()
	close(sigs)
	close(done)
	fmt.Printf("%d tasks finished\n", count)
}
