package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"prototest/pt"
)

// Data 구조체 정의
type Data struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Sex     string `json:"sex"`
}

func main() {
	url := flag.String("rx_url", "", "Rx Server URL")
	flag.Parse()

	// Tx 서버 url이 설정되지 않은 경우 -> 오류 출력
	if *url == "" {
		fmt.Println("Error: Tx server url must be specified.")
		os.Exit(1)
	}

	// 주기적으로 GET 요청 보내기
	for {
		sendGetRequest(*url)
		time.Sleep(time.Duration(10 * time.Second))
	}
}

func sendGetRequest(url string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error sending GET request to %s: %v\n", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Rx server responded with status code: %d\n", resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}

	var data []Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Printf("Error parsing response JSON: %v\n", err)
		return
	}

	fmt.Println("Received data from Rx server:")
	for _, d := range data {
		fmt.Printf("ID: %d, Name: %s, Address: %s, Sex: %s\n", d.Id, d.Name, d.Address, d.Sex)
	}
}
