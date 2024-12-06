package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"prototest/pt"
)

type Data struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Sex     string `json:"sex"`
}

// n만큼 데이터 생성
func generateData(n int) []Data {
	data := make([]Data, n)
	for i := 1; i <= n; i++ {
		data[i-1] = Data{
			Id:      i,
			Name:    fmt.Sprintf("Alex%d", i),
			Address: "123 Main Street",
			Sex:     "Male",
		}
	}
	return data
}

// 서버로 요청 보내기
func sendRequest(method, url string, data Data) error {
	client := &http.Client{}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	return nil
}

func main() {
	n := flag.Int("n", 1, "Number of data to generate")
	method := flag.String("m", "POST", "Request Method to Server (POST, PUT, DELETE)")
	url := flag.String("tx_url", "", "Tx Server URL")
	flag.Parse()

	// 유효성 검사
	if *n <= 0 {
		fmt.Println("Number of data entries (n) must be greater than 0")
		os.Exit(1)
	}

	// Tx 서버 url이 설정되지 않은 경우 -> 오류 출력
	if *url == "" {
		fmt.Println("Error: Tx server url must be specified.")
		os.Exit(1)
	}

	// 데이터 생성 및 전송
	num := generateData(*n)
	for _, data := range num {
		err := sendRequest(*method, *url, data)
		if err != nil {
			fmt.Printf("Error sending request for ID %d: %v\n", data.Id, err)
		}
	}

	// 사용자 실시간 입력 지원
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Enter a method (POST,PUT,DELETE) or type 'exit' to quit:")
		fmt.Print(">> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "exit" {
			fmt.Println("Exiting client...")
			break
		}

		// 유효한 메서드인지 확인
		if input != "POST" && input != "PUT" && input != "DELETE" {
			fmt.Println("Invalid method. Please enter POST, PUT, or DELETE.")
			continue
		}

		// 요청 데이터 반복 전송
		for _, data := range num {
			err := sendRequest(input, *url, data)
			if err != nil {
				fmt.Printf("Error sending request for ID %d: %v\n", data.Id, err)
			}
		}
	}
}
