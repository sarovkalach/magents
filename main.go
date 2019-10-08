package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type matchCounter struct {
	totalCounter int32
	regexp       *regexp.Regexp
	input        *bufio.Reader
	pool         chan string
	done         chan bool
}

func newMatchCounter(str string, maxWorkers int) *matchCounter {
	reg, _ := regexp.Compile(str)
	reader := bufio.NewReader(os.Stdin)
	urlPool := make(chan string, maxWorkers)
	doneCh := make(chan bool, maxWorkers)

	return &matchCounter{
		regexp: reg,
		input:  reader,
		pool:   urlPool,
		done:   doneCh,
	}
}

func (mc *matchCounter) ReadInput() {
	for {
		url, err := mc.input.ReadString('\n')
		if err != nil {
			break
		}
		url = strings.TrimSuffix(url, "\n")
		// fmt.Println("URL READED:", url)
		mc.pool <- url
	}
	close(mc.pool)
}

func (mc *matchCounter) request(url string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	strBody := string(body)
	instances := len(mc.regexp.FindAllString(strBody, -1))
	atomic.AddInt32(&mc.totalCounter, int32(instances))

	fmt.Printf("Count for %s: %d %d\n", url, instances, mc.totalCounter)

	<-mc.done
	return nil
}

func main() {
	mc := newMatchCounter("Go", 5)
	go func() {
		mc.ReadInput()
	}()

	wg := &sync.WaitGroup{}

	for url := range mc.pool {
		wg.Add(1)
		url := url
		mc.done <- true
		go func() {
			// fmt.Println("START:", url)
			mc.request(url)
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println("Total:", mc.totalCounter)
}
