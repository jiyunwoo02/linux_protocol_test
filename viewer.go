package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type vData struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Sex     string `json:"sex"`
}

func main() {
	url := flag.String("rx_url", "", "Rx Server URL")
	flag.Parse()

	if *url == "" {
		fmt.Println("Error: Rx server url must be specified.")
		os.Exit(1)
	}

	for {
		sendGetRequest(*url)
		time.Sleep(10 * time.Second)
	}
}

func sendGetRequest(url string) {
	// 기본적으로 Go의 http 클라이언트는 자체 서명된 인증서 신뢰 X -> tls: bad certificate 오류 발생
	tr := &http.Transport{
		// If InsecureSkipVerify is true,
		// crypto/tls accepts any certificate presented by the server and any host name in that certificate.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // (자체 서명) SSL 인증서 검증 생략
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error sending GET request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// HTTP 응답의 JSON 데이터를 읽어와 바이트 슬라이스(body)로 저장
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}

	// HTTP 응답의 body에서 가져온 JSON 데이터를
	// -> data 구조체 슬라이스로 언마샬링(역직렬화)
	var data []vData
	err = json.Unmarshal(body, &data) // &data: data의 포인터, 포인터 전달해 Unmarshal 함수는 data 직접 수정
	if err != nil {
		fmt.Printf("Error parsing response JSON: %v\n", err)
		return
	}

	fmt.Println("GET data from Rx:")
	for _, d := range data {
		log.Printf("ID: %d, Name: %s, Address: %s, Sex: %s\n", d.Id, d.Name, d.Address, d.Sex)
	}
}
