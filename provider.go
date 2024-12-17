package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultAddress = "123 Main Street"
	DefaultSex     = "Male"
)

type pData struct {
	// 각 필드가 JSON으로 변환될 때 어떤 키로 매핑되는지 명시
	// + 만약 JSON 태그를 생략하면, 구조체 필드 이름이 그대로 JSON 키로 사용
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Sex     string `json:"sex"`
}

// -pro=https인 경우 대비
var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // SSL 인증서 검증 생략
	},
}

func generateData(n int) []pData {
	// []pData: pData 구조체 타입의 슬라이스 (크기 가변적) -> 여러 개의 구조체 담기
	/* 예시
	data1 := pData{Id:1, Name:"Alex1", Address: "123 Main Street", Sex:"Male"}
	data2 := pData{Id:2, Name:"Alex2", Address: "123 Main Street", Sex:"Male"}
	dataList := []pData{data1, data2}
	fmt.Println(dataList) => [{1 Alex1 123 Main Street Male} {2 Alex2 456 Elm Street Female}]
	*/
	data := make([]pData, n)
	for i := 1; i <= n; i++ {
		data[i-1] = pData{
			Id:      i,
			Name:    fmt.Sprintf("Alex%d", i),
			Address: DefaultAddress,
			Sex:     DefaultSex,
		}
	}
	return data
}

// 요청을 보낼 때 -> 데이터 목록 전체를 JSON 배열로 묶어 한 번에 보내도록 수정!
func sendRequest(method, url string, data []pData) error {
	// 배열을 JSON 형식으로 직렬화
	// -> 구조체의 필드에 설정된 JSON 태그(json:"key")를 기반으로 JSON 키와 값을 매칭
	/*
			[
		  	  {
			    "id": 1,
			    "name": "Alex1",
			    "address": "123 Main Street",
			    "sex": "Male"
			  },
			  {
			    "id": 2,
			    "name": "Alex2",
			    "address": "456 Elm Street",
			    "sex": "Female"
			  }
			]
	*/
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	// JSON 데이터를 HTTP 요청 본문으로 추가: http.NewRequest는 https 지원
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	//💡서버에게 요청 본문이 JSON 형식임을 알림 (필수 X)(명확성 -> 서버와의 원활한 의사소통, 에러 발생 가능성 감소)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()
	return nil
}

func main() {
	method := flag.String("m", "", "Request Method to Server (POST, PUT, DELETE)")
	url := flag.String("tx_url", "", "Tx Server URL")
	// 메서드에 따라서 추가 명령행 인자
	n := flag.Int("n", 0, "Number of data to generate (for POST)")
	id := flag.Int("id", 0, "ID of the entry (for PUT/DELETE)")
	name := flag.String("name", "", "Name to update (for PUT)")
	flag.Parse()

	if *url == "" {
		fmt.Println("Error: Tx server url must be specified.")
		os.Exit(1)
	}

	switch strings.ToUpper(*method) {
	case "POST":
		if *n <= 0 {
			fmt.Println("Error: Number of data entries must be greater than 0.")
			os.Exit(1)
		}
		dataList := generateData(*n)
		// POST 요청을 한 번에 전체 데이터 배열로 보냄
		err := sendRequest("POST", *url, dataList)
		if err != nil {
			fmt.Printf("Error sending request: %v\n", err)
		}
	case "PUT":
		if *id <= 0 || *name == "" {
			fmt.Println("Error: PUT requires id and name.")
			os.Exit(1)
		}
		data := pData{
			Id:      *id,
			Name:    *name,
			Address: DefaultAddress,
			Sex:     DefaultSex,
		}
		err := sendRequest("PUT", *url, []pData{data})
		if err != nil {
			fmt.Printf("Error sending PUT request: %v\n", err)
		}
	case "DELETE":
		if *id <= 0 {
			fmt.Println("Error: DELETE requires id.")
			os.Exit(1)
		}
		data := pData{ // data는 pData 타입의 단일 구조체
			Id: *id,
		}
		err := sendRequest("DELETE", *url, []pData{data}) //  []pData{data}: 해당 구조체를 하나의 요소로 가진 슬라이스
		if err != nil {
			fmt.Printf("Error sending DELETE request for ID %d: %v\n", data.Id, err)
		}
	default:
		fmt.Println("Error: Invalid method. Use POST, PUT, or DELETE.")
		os.Exit(1)
	}

	// 명령행 인자로 지정한 작업 진행 후, 메서드 입력하도록
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Type a method (POST,PUT,DELETE) or 'exit' to quit.")
		fmt.Print(">> ")

		// ReadString reads until the first occurrence of delim in the input
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input. Please try again.")
			continue // for문 처음으로 돌아가 재시작
		}
		input = strings.TrimSpace(input)
		if strings.ToLower(input) == "exit" {
			fmt.Println("Exiting client...")
			break // 루프 종료
		}

		if strings.ToUpper(input) == "POST" {
			fmt.Print("Enter number of data to generate (n): ")
			nStr, _ := reader.ReadString('\n')
			nStr = strings.TrimSpace(nStr)
			n, err := strconv.Atoi(nStr)
			// 숫자 사이에 공백이 들어가면? -> 숫자 사이에 공백이 들어간 하나의 문자열로 처리, 유효한 숫자 형식이 아니어서 에러 반환
			if err != nil || n <= 0 {
				fmt.Println("Invalid number.")
				continue
			}
			dataList := generateData(n)
			// POST 요청을 한 번에 전체 데이터 배열로 보냄
			err = sendRequest("POST", *url, dataList)
			if err != nil {
				fmt.Printf("Error sending request: %v\n", err)
			}
		} else if strings.ToUpper(input) == "PUT" {
			fmt.Print("Enter ID to update: ")
			idStr, _ := reader.ReadString('\n')
			idStr = strings.TrimSpace(idStr)
			id, err := strconv.Atoi(idStr)
			if err != nil || id <= 0 {
				fmt.Println("Invalid ID.")
				continue
			}
			fmt.Print("Enter new name: ")
			name, _ := reader.ReadString('\n')
			name = strings.TrimSpace(name)
			if name == "" {
				fmt.Println("Name cannot be empty.")
				continue
			}
			data := pData{
				Id:      id,
				Name:    name,
				Address: DefaultAddress,
				Sex:     DefaultSex,
			}
			err = sendRequest("PUT", *url, []pData{data})
			if err != nil {
				fmt.Printf("Error sending PUT request: %v\n", err)
			}
		} else if strings.ToUpper(input) == "DELETE" {
			fmt.Print("Enter ID to delete: ")
			idStr, _ := reader.ReadString('\n')
			idStr = strings.TrimSpace(idStr)
			id, err := strconv.Atoi(idStr)
			if err != nil || id <= 0 {
				fmt.Println("Invalid ID.")
				continue
			}
			data := pData{
				Id: id,
			}
			err = sendRequest("DELETE", *url, []pData{data})
			if err != nil {
				fmt.Printf("Error sending DELETE request for ID %d: %v\n", data.Id, err)
			}
		} else {
			fmt.Println("Error: Invalid method. Use POST, PUT, or DELETE.")
			os.Exit(1)
		}
	}
}
