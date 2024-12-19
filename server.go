package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"prototest/pt"
	"time"

	//"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	httpPort  = "8080"
	httpsPort = "8443"
	tcpPort   = "1884"
)

type sData struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Sex     string `json:"sex"`
}

var TxData []*pt.Data
var RxData []*pt.Data

// var rxDataMutex sync.RWMutex

func main() {
	mode := flag.String("mode", "tx", "tx(transport) or rx(receive)")
	protocol := flag.String("pro", "http", "http or https")
	flag.Parse()

	if *mode == "tx" {
		startTxServer(*protocol)
	} else if *mode == "rx" {
		startRxServer(*protocol)
	} else {
		fmt.Println("tx와 rx 중 입력 바람")
		os.Exit(1)
	}
}

func startTxServer(protocol string) {
	if protocol == "http" {
		log.Printf("Starting HTTP Tx server on port %s", httpPort)
		http.HandleFunc("/", handleTxRequest)                          // 요청 처리 함수 설정
		if err := http.ListenAndServe(":"+httpPort, nil); err != nil { // HTTP 서버 실행
			log.Fatalf("Failed to start HTTP Tx server: %v", err)
		}
	} else if protocol == "https" {
		log.Printf("Starting HTTPS Tx server on port %s", httpsPort)
		http.HandleFunc("/", handleTxRequest)
		if err := http.ListenAndServeTLS(":"+httpsPort, "cert.pem", "key.pem", nil); err != nil {
			log.Fatalf("Failed to start HTTPS Tx server: %v", err)
		}
	} else {
		log.Print("http와 https 중 입력 바람")
		os.Exit(1)
	}
}

func startRxServer(protocol string) {
	if protocol == "http" {
		log.Printf("Starting HTTP Rx server on port %s", httpPort)
		go startRxTcpServer() // tcp 소켓으로부터 데이터 수신하도록
		http.HandleFunc("/", handleRxRequest)
		if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
			log.Fatalf("Failed to start HTTP Rx server: %v", err)
		}
	} else if protocol == "https" {
		log.Printf("Starting HTTPS Rx server on port %s", httpsPort)
		go startRxTcpServer()
		http.HandleFunc("/", handleRxRequest)
		if err := http.ListenAndServeTLS(":"+httpsPort, "cert.pem", "key.pem", nil); err != nil {
			log.Fatalf("Failed to start HTTPS Rx server: %v", err)
		}
	} else {
		log.Print("http와 https 중 입력 바람")
		os.Exit(1)
	}
}

func handleTxRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		start := time.Now()
		responseData, err := json.Marshal(TxData)
		if err != nil {
			log.Printf("Failed to marshal Tx data: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseData)
		end := time.Since(start)
		//log.Println("Tx - Processed GET request")
		fmt.Printf("-- Tx_Time elapsed for GET request: %d ms.\n", end.Milliseconds())
	} else if r.Method == http.MethodPost {
		processTxData(r, "POST")
		//log.Println("Tx - Processed POST request")
	} else if r.Method == http.MethodPut {
		processTxData(r, "PUT")
		//log.Println("Tx - Processed PUT request")
	} else if r.Method == http.MethodDelete {
		processTxData(r, "DELETE")
		//log.Println("Tx - Processed DELETE request")
	} else {
		log.Println("Method not allowed")
	}
}

func handleRxRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		start := time.Now()
		responseData, err := json.Marshal(RxData)
		if err != nil {
			log.Printf("Failed to marshal Rx data: %v", err)
			return // 에러가 발생하면 함수 종료
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseData)
		end := time.Since(start)
		log.Println("Rx - Processed GET request")
		fmt.Printf("-- Rx_Time elapsed for GET request: %d ms.\n", end.Milliseconds())
	} else {
		log.Println("Method not allowed")
	}
}

func processTxData(r *http.Request, method string) {
	// 여러 개의 데이터를 처리하도록 수정 (슬라이스 적용)
	var dataList []sData                                              // 클라이언트가 보낸 데이터 목록 -> JSON으로 디코딩된 구조체(sData) 형태
	if err := json.NewDecoder(r.Body).Decode(&dataList); err != nil { // HTTP 요청의 본문 (r.Body)에서 데이터를 읽어와서 dataList 변수에 파싱
		log.Printf("Invalid data format: %v", err)
		return
	}

	if method == "POST" {
		start := time.Now()
		// 받은 데이터를 TxData로 덮어쓰기 (기존 데이터는 모두 삭제)
		var txList []*pt.Data
		for _, data := range dataList {
			// dataList를 순회하며 각 구조체 요소를 *pt.Data 프로토버프 형삭으로 변환.
			txProtobuf := &pt.Data{
				Id:      int32(data.Id),
				Name:    data.Name,
				Address: data.Address,
				Sex:     data.Sex,
			}
			txList = append(txList, txProtobuf)
		}
		TxData = txList // TxData를 새로 받은 데이터로 교체
		end := time.Since(start)
		log.Printf("POST request processed for %d data.\n", len(dataList))
		fmt.Printf("-- Tx_Time elapsed for POST request: %d ms.\n", end.Milliseconds())
	}

	if method == "PUT" {
		start := time.Now()
		for _, data := range dataList {
			found := false
			for i, existingData := range TxData { // i는 현재 항목의 인덱스, existingData는 그 항목의 값
				if existingData.Id == int32(data.Id) {
					// 기존 Tx 데이터 갱신
					TxData[i] = &pt.Data{
						Id:      int32(data.Id),
						Name:    data.Name,
						Address: data.Address,
						Sex:     data.Sex,
					}
					found = true
					break
				}
			}
			end := time.Since(start)
			if !found {
				log.Printf("PUT request: ID %d not found, skipping update.\n", data.Id)
			} else {
				log.Printf("PUT request processed for ID %d.\n", data.Id)
				fmt.Printf("-- Tx_Time elapsed for POST request: %d ms.\n", end.Milliseconds())
			}
		}
	}

	if method == "DELETE" {
		start := time.Now()
		for _, data := range dataList {
			found := false
			for i, existingData := range TxData {
				if existingData.Id == int32(data.Id) {
					// 슬라이스에서 해당 데이터 삭제
					TxData = append(TxData[:i], TxData[i+1:]...) // 0번째부터 i-1번째, i+1번째부터 마지막까지의 모든 요소 합치기
					found = true
					break
				}
			}
			end := time.Since(start)
			if !found {
				log.Printf("DELETE request: ID %d not found, skipping deletion.\n", data.Id)
			} else {
				log.Printf("DELETE request processed for ID %d.\n", data.Id)
				fmt.Printf("-- Tx_Time elapsed for POST request: %d ms.\n", end.Milliseconds())
			}
		}
	}
	log.Printf("Current TxData: %+v\n", TxData)

	// Rx 서버로 데이터 패키지 전송
	dataPackage := &pt.DataPackage{
		DataList:   TxData,             // 여러 개의 pt.Data 구조체를 가진 슬라이스
		TotalCount: int32(len(TxData)), //  TxData에 포함된 데이터 항목의 개수
	}
	err := sendToRx(dataPackage)
	if err != nil {
		log.Printf("Error sending data to Rx server: %v", err)
	}
}

func sendToRx(dataPackage *pt.DataPackage) error {
	// TCP 연결 설정
	conn, err := net.Dial("tcp", "localhost:"+tcpPort)
	if err != nil {
		return fmt.Errorf("failed to connect to Rx server: %w", err)
	}
	defer conn.Close()

	// Protocol Buffers 직렬화: Protobuf 객체를 바이트 배열로 변환
	// -> data 변수에는 Protobuf 포맷으로 인코딩된 데이터가 담김
	// -- 네트워크를 통해 데이터를 전송하려면 데이터를 바이트 스트림 형식으로 변환!
	data, err := proto.Marshal(dataPackage)
	if err != nil {
		return fmt.Errorf("failed to marshal data package: %w", err)
	}

	// 데이터 전송 전, "전송하려는 데이터(data)의 전체 길이" 정보 전송 -> 수신 측 socket buffer size 고려
	// 엔디안 사용 -> 서로 다른 시스템 간 데이터 전송에서 바이트 순서를 맞추기 위함
	length := int32(len(data))                            // int32는 4바이트(32비트) 크기 가짐
	lengthBuf := make([]byte, 4)                          // 4바이트 버퍼 생성
	binary.BigEndian.PutUint32(lengthBuf, uint32(length)) // Big Endian 방식으로 버퍼에 길이를 기록 (엔디안: 컴퓨터가 메모리에 데이터를 읽고 쓰는 방향을 정의)
	if _, err := conn.Write(lengthBuf); err != nil {
		return fmt.Errorf("failed to send data length to Rx server: %w", err)
	}

	// log.Printf("Tx server sent %d bytes to Rx server. (protobuf) \n", length)

	// 데이터 전송: conn.Write(data)가 항상 전체 데이터를 한 번에 전송한다고 보장할 수 없다
	// 네트워크가 안정적이라면 length와 bytesSent 동일, 불안정하거나 버퍼 크기 작다면 상이
	// -> 두 번의 write가 하나의 연속적인 데이터 스트림을 형성
	start := time.Now()
	bytesSent, err := conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send data to Rx server: %w", err)
	}
	end := time.Since(start)

	// "실제로 전송된 데이터의 크기" 출력
	log.Printf("Tx server sent %d bytes to Rx server. (protobuf) \n", bytesSent)
	// 소요 시간 출력
	fmt.Printf("-- Tx_Time elapsed for Socket Sending: %d ms.\n", end.Milliseconds())
	return nil

	// log.Printf("Data sent to Rx server: %+v\n", dataPackage)
	/*
		pt.DataPackage{
		    DataList: []*pt.Data{
		        &pt.Data{Id: 1, Name: "Alice", Address: "123 Maple St", Sex: "Female"},
		        &pt.Data{Id: 2, Name: "Bob", Address: "456 Oak St", Sex: "Male"},
		        &pt.Data{Id: 3, Name: "Charlie", Address: "789 Pine St", Sex: "Male"},
		    },
		    TotalCount: 3,
		}
	*/
}

func startRxTcpServer() {
	listener, err := net.Listen("tcp", ":"+tcpPort)
	if err != nil {
		log.Fatalf("Failed to start Rx TCP server: %v", err)
	}
	defer listener.Close()

	log.Printf("Rx TCP server started on port %s\n", tcpPort)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleRxConn(conn) // 각 클라이언트와의 연결을 병렬로 처리
	}
}

func handleRxConn(conn net.Conn) {
	start := time.Now()
	defer conn.Close()

	// 데이터 길이 수신
	// -> 4바이트로 설정하지 않으면 Error reading from connection: EOF
	// make([]byte, 10): 실제로 길이 정보는 4바이트밖에 전송되지 않으므로, 나머지 6바이트에 대해 데이터를 찾을 수 없어 EOF 오류가 발생
	lengthBuf := make([]byte, 4)
	if _, err := conn.Read(lengthBuf); err != nil {
		log.Printf("Error reading data length from connection: %v", err)
		return
	}
	dataLength := binary.BigEndian.Uint32(lengthBuf)
	// log.Printf("Rx server received %d bytes from Tx server. (protobuf) \n", dataLength)

	// 데이터 수신
	buf := make([]byte, dataLength)
	totalRead := 0
	for totalRead < int(dataLength) {
		n, err := conn.Read(buf[totalRead:]) // 반복적으로 Read할 수 있도록
		if err != nil {
			log.Printf("Error reading from connection: %v", err)
			return
		}
		totalRead += n // 읽은 바이트 수를 기록
	}
	end := time.Since(start)
	fmt.Printf("-- Rx_Time elapsed for Socket Receiving: %d ms.\n", end.Milliseconds())

	// Protobuf 메시지 디코딩: 네트워크를 통해 수신한 바이트 데이터를 Protobuf 객체로 디코딩
	var dataPackage pt.DataPackage
	if err := proto.Unmarshal(buf, &dataPackage); err != nil {
		log.Printf("Error unmarshaling protobuf data: %v", err)
		return
	}

	// TotalCount vs 수신 데이터의 개수
	if int(dataPackage.TotalCount) == len(dataPackage.DataList) {
		// 개수 일치 -> Tx에서 송신한 데이터를 Rx에 반영
		log.Printf("Data count matches, updating RxData with received data.")
		RxData = dataPackage.DataList
	} else {
		// 개수 불일치 -> 기존 RxData 유지
		log.Printf("Data count mismatch, keeping current RxData.")
	}

	// Protobuf 객체를 JSON으로 변환
	jsonData, err := protojson.Marshal(&dataPackage)
	if err != nil {
		log.Printf("Error converting protobuf to JSON: %v", err)
		return
	}

	// 수신된 데이터 바이트 수 출력
	log.Printf("Rx server received %d bytes from Tx server. (protobuf) \n", totalRead)
	log.Printf("Rx server received data: %s", string(jsonData))

	//rxDataMutex.Lock()
	//rxDataMutex.Unlock()
}

// 서버가 종료될 때 모든 고루틴이 종료될 때까지 기다려야 하는 경우 -> 웨이트그룹 사용
