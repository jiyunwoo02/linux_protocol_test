package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"prototest/pt"
	"sync"

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

// pt.Data 구조체 포인터의 슬라이스, 내부에 pt.Data 구조체의 메모리 주소(포인터) 저장
var TxData []*pt.Data
var rxData []*pt.Data
var rxDataMutex sync.RWMutex

// Rx 서버가 TCP 연결을 수신하고 처리하기 위해 동작하는 함수
func receivefromTx(tcpPort string) {
	listener, err := net.Listen("tcp", ":"+tcpPort)
	if err != nil {
		log.Fatalf("Failed to start TCP listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Rx server listening on TCP port %s", tcpPort)
	for {
		// 클라이언트가 연결 요청을 보낼 때까지 대기
		// 연결 요청이 성공하면, 새로 생성된 net.Conn 객체 반환 및 이를 conn 변수에 저장
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue // 실패한 연결 건너뛰고 다음 요청을 대기
		}
		go handleConnection(conn)
	}
}

// TCP 연결을 통해 수신한 데이터를 처리하는 함수
func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 5000) // 데이터 저장할 버퍼 생성
	n, err := conn.Read(buffer)  // TCP 연결에서 데이터 읽기, n: 읽은 데이터의 크기
	if err != nil {
		log.Printf("Failed to read data: %v", err)
		return
	}

	var receivedPackage pt.DataPackage
	err = proto.Unmarshal(buffer[:n], &receivedPackage) // 실제로 읽은 데이터를 Protobuf 형식으로 역직렬화해 receivedPackage에 저장
	if err != nil {
		log.Printf("Failed to deserialize data: %v", err)
		return
	}

	// Rx 서버가 수신한 데이터의 개수가 TotalCount와 일치하는지 검증
	if len(receivedPackage.DataList) != int(receivedPackage.TotalCount) {
		log.Printf("Mismatch in data count: expected %d, got %d", receivedPackage.TotalCount, len(receivedPackage.DataList))
		return
	}

	// ... : 가변 인수 함수 호출 시 사용 -> convertToList로 반환된 배열을 하나씩 분리하여 append에 전달
	// convertToList가 [a, b, c] 배열을 반환하면, append(rxData, a, b, c)처럼 각 요소를 개별 인수로 전달
	rxDataMutex.Lock()
	rxData = append(rxData, convertToList(receivedPackage.DataList)...)
	rxDataMutex.Unlock()
}

func convertToList(protoList []*pt.Data) []*pt.Data {
	// *pt.Data 타입의 포인터 배열 protoList를 받아서,
	// 그 배열을 복사하고 새로운 배열을 만들어 반환
	
	// dataList := make([]*pt.Data, len(protoList))
	for i, d := range protoList {
		// dataList[i] = &pt.Data{
		protoList[i] = &pt.Data{
			Id:      d.Id,
			Name:    d.Name,
			Address: d.Address,
			Sex:     d.Sex,
		}
	}
	// return dataList // -> 원본 데이터는 그대로 두고 복사본을 수정
	return protoList
}

// 주어진 dataList를 pt.DataPackage 구조체로 변환하여 반환
func convertToPackage(dataList []*pt.Data) *pt.DataPackage {
	// & 사용 -> pt.DataPackage 구조체 전체를 복사하지 않고, 구조체의 메모리 주소만 반환
	return &pt.DataPackage{
		DataList:   dataList,
		TotalCount: int32(len(dataList)),
	}
}

// dataList를 받아서 Protobuf 형식으로 직렬화한 후, TCP 연결을 통해 rxurl의 Rx 서버로 전송
func sendtoRx(rxurl string, dataList []*pt.Data) {
	// Protobuf 직렬화
	dataPackage := convertToPackage(dataList)
	dataBytes, err := proto.Marshal(dataPackage)
	if err != nil {
		log.Fatalf("Failed to serialize data: %v", err)
	}

	// TCP 연결 설정
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
}

func startHTTPServer(role string) {
	// HTTP 요청을 처리할 멀티플렉서(mux) 생성
	mux := http.NewServeMux()

	if role == "Tx" {
		// "/" 경로에 대한 HTTP 요청을 처리하는 핸들러 설정
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handleTxRequest(r, w, "localhost:"+tcpPort)
		})
	} else if role == "Rx" {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handleRxRequest(r, w)
		})
	}

	log.Printf("Starting HTTP server for %s on port %s", role, httpPort) // 시간 정보 출력 - log
	http.ListenAndServe(":"+httpPort, mux)
}

func startHTTPSServer(role string) {
	mux := http.NewServeMux()

	if role == "Tx" {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handleTxRequest(r, w, "localhost:"+tcpPort)
		})
	} else if role == "Rx" {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handleRxRequest(r, w)
		})
	}

	log.Printf("Starting HTTPS server for %s on port %s", role, httpsPort)
	http.ListenAndServeTLS(":"+httpsPort, "cert.pem", "key.pem", mux)
}

func handleTxRequest(r *http.Request, w http.ResponseWriter, rxurl string) {
	switch r.Method {
	case http.MethodPost:
		processTxData(r, w, rxurl, "POST")
	case http.MethodPut:
		processTxData(r, w, rxurl, "PUT")
	case http.MethodDelete:
		processTxData(r, w, rxurl, "DELETE")
	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

func processTxData(r *http.Request, w http.ResponseWriter, rxAddr string, method string) {
	var data sData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 데이터를 Protobuf 형식으로 변환 및 전송
	protoData := &pt.Data{
		Id:      int32(data.Id),
		Name:    data.Name,
		Address: data.Address,
		Sex:     data.Sex,
	}

	// Rx 서버로 데이터 전송
	switch method {
	case "POST":
		sendtoRx(rxAddr, []*pt.Data{protoData})
		log.Printf("Tx processed POST request for ID %d", data.Id)
	case "PUT":
		sendtoRx(rxAddr, []*pt.Data{protoData})
		log.Printf("Tx processed PUT request for ID %d", data.Id)
	case "DELETE":
		sendtoRx(rxAddr, []*pt.Data{protoData})
		log.Printf("Tx processed DELETE request for ID %d", data.Id)
	}

	printStoredData()
	w.WriteHeader(http.StatusOK) // 응답 상태 코드 200 OK
}

func handleRxRequest(r *http.Request, w http.ResponseWriter) {
	switch r.Method {
	case http.MethodPost:
		processRxData(r, "POST")
	case http.MethodPut:
		processRxData(r, "PUT")
	case http.MethodDelete:
		processRxData(r, "DELETE")
	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

func processRxData(r *http.Request, method string) {
	var data sData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Printf("Invalid data format: %v", err)
		return
	}

	switch method {
	case "POST":
		replaced := false
		for i, d := range rxData {
			if d.Id == int32(data.Id) {
				// 데이터 덮어쓰기
				rxData[i] = &pt.Data{
					Id:      int32(data.Id),
					Name:    data.Name,
					Address: data.Address,
					Sex:     data.Sex,
				}
				replaced = true
				log.Printf("POST - Data with ID %d replaced", data.Id)
				break
			}
		}
		if !replaced {
			// 새 데이터 추가
			rxData = append(rxData, &pt.Data{
				Id:      int32(data.Id),
				Name:    data.Name,
				Address: data.Address,
				Sex:     data.Sex,
			})
			log.Printf("POST - Data with ID %d added", data.Id)
		}
		printStoredData() // 데이터 출력

	case "PUT":
		// 지정한 아이디에 해당하는 이름 필드 수정
		updated := false
		for _, d := range rxData {
			if d.Id == int32(data.Id) {
				d.Name = data.Name
				d.Address = data.Address
				d.Sex = data.Sex
				updated = true
				log.Printf("-- Rx updated data with ID %d", data.Id)
				break
			}
		}
		if !updated {
			log.Printf("No data found with ID %d to update", data.Id)
		}

	case "DELETE":
		// 지정한 아이디에 해당하는 데이터 삭제
		deleted := false
		for i, d := range rxData {
			if d.Id == int32(data.Id) {
				rxData = append(rxData[:i], rxData[i+1:]...) // 해당 데이터를 삭제
				deleted = true
				log.Printf("-- Rx deleted data with ID %d", data.Id)
				break
			}
		}
		if !deleted {
			log.Printf("No data found with ID %d to delete", data.Id)
		}
		printStoredData()
	}
}

// 저장된 데이터를 출력하는 함수
func printStoredData() {
	if len(rxData) == 0 {
		log.Println("No data stored")
		return
	}

	log.Println("Current stored data in Rx:")
	for _, data := range rxData {
		log.Printf("ID: %d, Name: %s, Address: %s, Sex: %s", data.Id, data.Name, data.Address, data.Sex)
	}
}

func main() {
	txProtocol := flag.String("tx", "", "Protocol for Tx (http or https)")
	rxProtocol := flag.String("rx", "", "Protocol for Rx (http or https)")
	flag.Parse()

	if *txProtocol == "" || *rxProtocol == "" {
		fmt.Println("Error: Both Tx and Rx protocols must be specified.")
		os.Exit(1)
	}
	if (*txProtocol != "http" && *txProtocol != "https") || (*rxProtocol != "http" && *rxProtocol != "https") {
		fmt.Println("Error: Protocol must be either 'http' or 'https'.")
		os.Exit(1)
	}
	if *txProtocol == *rxProtocol {
		fmt.Println("Error: Tx and Rx cannot use the same protocol.")
		os.Exit(1)
	}
	fmt.Printf("Starting server with Tx: %s and Rx: %s\n", *txProtocol, *rxProtocol)

	if *txProtocol == "http" {
		go startHTTPServer("Tx")
	} else {
		go startHTTPSServer("Tx")
	}
	if *rxProtocol == "http" {
		go receivefromTx(tcpPort)
		go startHTTPServer("Rx")
	} else {
		go receivefromTx(tcpPort)
		go startHTTPSServer("Rx")
	}
	select {}
}
