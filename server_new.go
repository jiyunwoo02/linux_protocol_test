package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"prototest/pt"
	"sync"
	// "google.golang.org/protobuf/proto"
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
var rxData []*pt.Data
var rxDataMutex sync.RWMutex

func main() {
	mode := flag.String("mode", "tx", "tx or rx")
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
		http.HandleFunc("/", handleTxRequest)
		if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
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
		http.HandleFunc("/", handleRxRequest)
		if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
			log.Fatalf("Failed to start HTTP Rx server: %v", err)
		}
	} else if protocol == "https" {
		log.Printf("Starting HTTPS Rx server on port %s", httpsPort)
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
	if r.Method == http.MethodPost {
		log.Println("Tx - Processing POST request")
		processTxData(r, "POST")
	} else if r.Method == http.MethodPut {
		log.Println("Tx - Processing PUT request")
		processTxData(r, "PUT")
	} else if r.Method == http.MethodDelete {
		log.Println("Tx - Processing DELETE request")
		processTxData(r, "DELETE")
	} else {
		log.Println("Method not allowed")
	}
}

func handleRxRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rxDataMutex.RLock()
		defer rxDataMutex.RUnlock()

		responseData, err := json.Marshal(rxData)
		if err != nil {
			log.Printf("Failed to marshal Rx data: %v", err)
			http.Error(w, "Failed to marshal data", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseData)
		log.Println("Rx - Processed GET request")
	} else {
		log.Println("Method not allowed")
	}
}

func processTxData(r *http.Request, method string) {
	var dataList []sData
	if err := json.NewDecoder(r.Body).Decode(&dataList); err != nil {
		log.Printf("Invalid data format: %v", err)
		return
	}

	// 데이터를 순차적으로 처리
	for _, data := range dataList {
		txProtobuf := &pt.Data{
			Id:      int32(data.Id),
			Name:    data.Name,
			Address: data.Address,
			Sex:     data.Sex,
		}

		if method == "POST" {
			// 중복된 아이디는 갱신, 새로운 아이디는 추가 -> 오류
			// 클라이언트가 데이터 1개마다 요청을 보내게 하지말고
			// 여러 개의 데이터를 1번의 요청으로 보내도록!
			TxData = nil
			TxData = append(TxData, txProtobuf)
		}

		if method == "PUT" {
			for i, data := range TxData {
				if data.Id == txProtobuf.Id {
					TxData[i] = txProtobuf
					break
				}
			}
			log.Printf("Tx - Processed PUT request for ID %d", data.Id)
		}

		if method == "DELETE" {
			for i, data := range TxData {
				if data.Id == txProtobuf.Id {
					TxData = append(TxData[:i], TxData[i+1:]...)
					break
				}
			}
			log.Printf("Tx - Processed DELETE request for ID %d", data.Id)
		}
		log.Printf("Current TxData: %+v", TxData)

		// 패키지로 만들어서 rx한테 보내자
		// dataPackage := &pt.DataPackage{
		// 	DataList:   TxData,
		// 	TotalCount: int32(len(TxData)),
		// }
		// if err := sendToRx(dataPackage); err != nil {
		// 	log.Printf("Error sending data to Rx server: %v", err)
		// }
	}
}
