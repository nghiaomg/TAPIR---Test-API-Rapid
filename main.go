package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"sort"
)

type Config struct {
	Method      string
	Workers     int
	TotalReqs   int
	BatchSize   int
	URL         string
	BodyFile    string
	Repeat      bool
	Duration    int
}

type Result struct {
	StatusCode int
	Response   string
	Count      int
}

func saveResponseToLog(statusCode int, response string) string {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return fmt.Sprintf("Error creating logs directory: %v", err)
	}

	timestamp := time.Now().UnixMilli()
	filename := fmt.Sprintf("Error%d_%d.txt", statusCode, timestamp)
	filepath := filepath.Join("logs", filename)

	err := ioutil.WriteFile(filepath, []byte(response), 0644)
	if err != nil {
		return fmt.Sprintf("Error writing to log file: %v", err)
	}

	return filename
}

func clearLine() {
	fmt.Printf("\r%s\r", strings.Repeat(" ", 150))
}

func main() {
	config := Config{}
	flag.StringVar(&config.Method, "m", "GET", "HTTP method (GET, POST, PUT, etc)")
	flag.IntVar(&config.Workers, "w", 100, "Number of concurrent workers")
	flag.IntVar(&config.TotalReqs, "n", 100000, "Total number of requests")
	flag.IntVar(&config.BatchSize, "batch", 1000, "Batch size")
	flag.StringVar(&config.URL, "url", "", "Target URL")
	flag.StringVar(&config.BodyFile, "b", "", "Body file path (JSON)")
	flag.BoolVar(&config.Repeat, "r", false, "Repeat mode")
	flag.IntVar(&config.Duration, "t", 0, "Duration in seconds for repeat mode")
	flag.Parse()

	if config.URL == "" {
		fmt.Println("URL is required")
		return
	}

	var bodyContent []byte
	var err error
	if config.BodyFile != "" {
		bodyContent, err = ioutil.ReadFile(config.BodyFile)
		if err != nil {
			fmt.Printf("Error reading body file: %v\n", err)
			return
		}
	}

	results := make(map[int]*Result)
	var resultMutex sync.Mutex
	var statusOrder []int
	processedCount := 0
	startTime := time.Now()

	resultChan := make(chan Result, config.BatchSize)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var timer *time.Timer
	if config.Repeat && config.Duration > 0 {
		timer = time.NewTimer(time.Duration(config.Duration) * time.Second)
		defer timer.Stop()
	}

	stopChan := make(chan bool)

	go func() {
		for result := range resultChan {
			resultMutex.Lock()
			if _, exists := results[result.StatusCode]; !exists {
				results[result.StatusCode] = &Result{
					StatusCode: result.StatusCode,
					Response:   result.Response,
					Count:      0,
				}
				statusOrder = append(statusOrder, result.StatusCode)
				sort.Ints(statusOrder)
			}
			results[result.StatusCode].Count++
			processedCount++

			select {
			case <-ticker.C:
				// Tính tỉ lệ thành công
				successRate := 0.0
				if processedCount > 0 {
					if successCount, ok := results[200]; ok {
						successRate = float64(successCount.Count) / float64(processedCount) * 100
					}
				}
				
				fmt.Print("\r\033[K")
				
				output := fmt.Sprintf("Processed: %d/%d (%.2f%% success)", 
					processedCount, config.TotalReqs, successRate)
				
				for _, statusCode := range statusOrder {
					if res, ok := results[statusCode]; ok {
						output += fmt.Sprintf(" | Status %d: %d/%d", 
							statusCode, res.Count, config.TotalReqs)
					}
				}
				
				elapsed := int(time.Since(startTime).Seconds())
				output += fmt.Sprintf(" | Time: %ds", elapsed)
				
				fmt.Print(output)
			default:
			}
			
			resultMutex.Unlock()
		}
	}()

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	clientPool := &sync.Pool{
		New: func() interface{} {
			return &http.Client{
				Transport: transport,
				Timeout:   30 * time.Second,
			}
		},
	}

	if config.Repeat && config.Duration > 0 {
		go func() {
			<-timer.C
			close(stopChan)
		}()
	}

	batchLoop:
	for {
		for i := 0; i < config.TotalReqs; i += config.BatchSize {
			select {
			case <-stopChan:
				break batchLoop
			default:
				batchSize := config.BatchSize
				if i+batchSize > config.TotalReqs {
					batchSize = config.TotalReqs - i
				}

				var wg sync.WaitGroup

				for j := 0; j < batchSize; j++ {
					wg.Add(1)
					go func(reqID int) {
						defer wg.Done()
						
						client := clientPool.Get().(*http.Client)
						defer clientPool.Put(client)

						makeRequest(config, bodyContent, reqID, resultChan, client)
						
						time.Sleep(1 * time.Millisecond)
					}(i + j)
				}

				wg.Wait()
				time.Sleep(100 * time.Millisecond)
			}
		}

		if !config.Repeat || config.Duration == 0 {
			break
		}
	}

	close(resultChan)

	fmt.Println("\n\nFinal Results:")
	for statusCode, result := range results {
		if statusCode != 200 {
			filename := saveResponseToLog(statusCode, result.Response)
			fmt.Printf("response %d: is saved in logs/%s\n", statusCode, filename)
		}
	}
	
	fmt.Printf("\nTotal time: %v\n", time.Since(startTime))
}

func makeRequest(config Config, body []byte, reqID int, resultChan chan Result, client *http.Client) {
	req, err := http.NewRequest(strings.ToUpper(config.Method), config.URL, strings.NewReader(string(body)))
	if err != nil {
		resultChan <- Result{StatusCode: 500, Response: err.Error()}
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		resultChan <- Result{StatusCode: 500, Response: err.Error()}
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		resultChan <- Result{StatusCode: 500, Response: err.Error()}
		return
	}

	resultChan <- Result{
		StatusCode: resp.StatusCode,
		Response:   string(respBody),
	}
} 