package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"prototest/pt"

	"google.golang.org/protobuf/proto"
)

const (
	httpPort  = "8080"
	httpsPort = "8443"
	tcpPort   = "1884"
)

var TxData []Data
var rxData []Data

// Data 구조체를 Protobuf의 Data 메시지 리스트로 변환
func convertToProtoDataList(dataList []Data) []*pt.Data {
	protoList := make([]*pt.Data, len(dataList))
	for i, d := range dataList {
		protoList[i] = &pt.Data{
			Id:      int32(d.Id),
			Name:    d.Name,
			Address: d.Address,
			Sex:     d.Sex,
		}
	}
	return protoList
}

func sendtoRx(rxurl string, data []Data) {
	// 데이터 변환 및 직렬화
	dataPackage := &pt.DataPackage{
		DataList:   convertToProtoDataList(data),
		TotalCount: int32(len(data)),
	}

	dataBytes, err := proto.Marshal(dataPackage)
	if err != nil {
		log.Fatalf("Failed to serialize data: %v", err)
	}

	// Rx 서버로 TCP 연결
	conn, err := net.Dial("tcp", rxurl)
	if err != nil {
		log.Fatalf("Failed to connect to Rx server: %v", err)
	}
	defer conn.Close()

	// 데이터 전송
	_, err = conn.Write(dataBytes)
	if err != nil {
		log.Fatalf("Failed to send data to Rx server: %v", err)
	}

	log.Println("Data successfully sent to Rx server")
}

// Rx 서버: TCP 소켓을 통해 데이터 수신
func receivefromTx(tcpPort string) {
	listener, err := net.Listen("tcp", ":"+tcpPort)
	if err != nil {
		log.Fatalf("Failed to start TCP listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Rx server listening on TCP port %s", tcpPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

// Rx 서버: TCP 연결 처리
func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 수신 데이터 읽기
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Failed to read data: %v", err)
		return
	}

	// Protobuf 데이터 역직렬화
	var receivedPackage pt.DataPackage
	err = proto.Unmarshal(buffer[:n], &receivedPackage)
	if err != nil {
		log.Printf("Failed to deserialize data: %v", err)
		return
	}

	// 데이터 처리
	log.Printf("Received %d data entries from Tx server", receivedPackage.TotalCount)
	for _, d := range receivedPackage.DataList {
		log.Printf("ID: %d, Name: %s, Address: %s, Sex: %s", d.Id, d.Name, d.Address, d.Sex)
	}

	// Rx 데이터에 저장
	rxData = append(rxData, convertToDataList(receivedPackage.DataList)...)
}

// Protobuf의 Data 메시지 리스트를 Go 구조체로 변환
func convertToDataList(protoList []*pt.Data) []Data {
	dataList := make([]Data, len(protoList))
	for i, p := range protoList {
		dataList[i] = Data{
			Id:      int(p.Id),
			Name:    p.Name,
			Address: p.Address,
			Sex:     p.Sex,
		}
	}
	return dataList
}

func main() {
	// 명령행 플래그 정의
	txProtocol := flag.String("tx", "", "Protocol for Tx (http or https)")
	rxProtocol := flag.String("rx", "", "Protocol for Rx (http or https)")
	flag.Parse()

	// Tx와 Rx가 설정되지 않은 경우 -> 오류 출력
	if *txProtocol == "" || *rxProtocol == "" {
		fmt.Println("Error: Both Tx and Rx protocols must be specified (http or https).")
		os.Exit(1)
	}

	// 유효한 프로토콜인지 확인
	if (*txProtocol != "http" && *txProtocol != "https") || (*rxProtocol != "http" && *rxProtocol != "https") {
		fmt.Println("Error: Protocol must be either 'http' or 'https'.")
		os.Exit(1)
	}

	// Tx와 Rx가 동일한 프로토콜을 사용하는 경우 -> 오류 처리
	if *txProtocol == *rxProtocol {
		fmt.Println("Error: Tx and Rx cannot use the same protocol.")
		os.Exit(1)
	}

	// 선택한 Tx와 Rx 프로토콜 출력
	fmt.Printf("Starting server with Tx: %s and Rx: %s\n", *txProtocol, *rxProtocol)

	if *txProtocol == "http" {
		go startHTTPServer("Tx")
		go sendtoRx("localhost:"+tcpPort, TxData)
	} else {
		go startHTTPSServer("Tx")
		go sendtoRx("localhost:"+tcpPort, TxData)
	}

	if *rxProtocol == "http" {
		go receivefromTx(tcpPort)
		startHTTPServer("Rx")
	} else {
		go receivefromTx(tcpPort)
		startHTTPSServer("Rx")
	}
}

// HTTP 서버 함수
func startHTTPServer(role string) {
	fmt.Printf("Starting HTTP server for %s...\n", role)
}

// HTTPS 서버 함수
func startHTTPSServer(role string) {
	fmt.Printf("Starting HTTPS server for %s...\n", role)
}
