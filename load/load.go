package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	wg := sync.WaitGroup{}
	for i := 0; i < 15; i++ {

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// var jsonStr = []byte(`{"url": "https://eu.httpbin.org/get"}`)
				// res, err := http.Post("http://localhost:5001/api/shorten", "application/json", bytes.NewBuffer(jsonStr))
				// if err != nil {
				// 	fmt.Println(err)
				// 	return
				// }
				// defer res.Body.Close()
				// bytedata, _ := io.ReadAll(res.Body)
				// print(string(bytedata))
				start := time.Now()
				res, err := http.Get("http://localhost:5001/c3c1b1e")
				if err != nil {
					fmt.Println(err)
					return
				}
				// bytedata, _ := io.ReadAll(res.Body)
				// print(string(bytedata))
				fmt.Printf("%d %.4fs\n", res.StatusCode, time.Since(start).Seconds())
			}()
		}
		wg.Wait()
	}

	// for i := 0; i < 10000; i++ {

	// 	// wg.Add(1)
	// 	// go func() {
	// 	// 	defer wg.Done()
	// 	body := fmt.Sprintf(`{"url": "https://eu%d.httpbin.org/get"}`, i)
	// 	var jsonStr = []byte(body)
	// 	res, err := http.Post("http://localhost:5001/api/shorten", "application/json", bytes.NewBuffer(jsonStr))
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}
	// 	defer res.Body.Close()
	// 	// }()

	// 	// wg.Wait()
	// }

}
