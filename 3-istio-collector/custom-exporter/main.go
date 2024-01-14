package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	logs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var podsMap = make(map[string]*corev1.Pod)

type myLog struct {
	Method   string `bson:"method"`
	Path     string `bson:"path"`
	Protocol string `bson:"protocol"`
	SrcIP    string `bson:"srcip"`
	SrcName  string `bson:"srcname"`
	DstIP    string `bson:"dstip"`
	DstName  string `bson:"dstname"`
}

type config struct {
	listenAddr string
	listenPort int
}

// LogServer is for exporting log handler
type LogServer struct {
	logs.UnimplementedLogsServiceServer
}

// Export interface function for LogServiceServer
func (ls LogServer) Export(c context.Context, request *logs.ExportLogsServiceRequest) (*logs.ExportLogsServiceResponse, error) {

	logText := request.String()

	// "default"를 기준으로 문자열 나누기
	parts := strings.Split(logText, "default\\")
	keywords := []string{"DELETE", "GET", "POST", "PATCH", "PUT"}
	var part2 string

	// 분할된 문자열에서 각각 마지막 3개의 단어 출력
	for _, part := range parts[0:] {
		var method string
		var path string
		var protocolName string

		for _, keyword := range keywords {
			if startIndex := strings.Index(part, keyword); startIndex != -1 {
				part2 = part[startIndex:]
				words := strings.Fields(part2)

				nextWords := words[:4] // startIndex부터 시작하여 4개의 단어 저장

				if len(nextWords) > 3 {
					method = nextWords[0]
					path = nextWords[1]
					protocolName = nextWords[2]
					protocolName = strings.ReplaceAll(protocolName, `"`, "")
					protocolName = strings.ReplaceAll(protocolName, `\`, "")
				}
			}
		}

		var srcIP string
		var dstIP string

		// 공백을 기준으로 나눈 후 마지막 3개 단어 출력
		words := strings.Fields(part)
		if len(words) >= 3 {
			lastThreeWords := words[len(words)-3 : len(words)]

			firstWord := lastThreeWords[0]
			secondWord := lastThreeWords[1]

			// ':' 문자를 찾아서 해당 문자 이전의 부분만 추출
			colonIndex := strings.LastIndex(firstWord, ":")
			if colonIndex > 0 {
				dstIP = strings.TrimSpace(firstWord[:colonIndex])
			}
			colonIndex2 := strings.LastIndex(secondWord, ":")
			if colonIndex2 > 0 {
				srcIP = strings.TrimSpace(secondWord[:colonIndex2])
			}
		}

		var sourcePodName string
		var destPodName string

		if pod, ok := podsMap[srcIP]; ok {
			sourcePodName = pod.Name
		} else {
			sourcePodName = "Not found"
		}

		if pod, ok := podsMap[dstIP]; ok {
			destPodName = pod.Name
		} else {
			destPodName = "Not found"
		}

		myLog := myLog{
			Method:   method,
			Path:     path,
			Protocol: protocolName,
			SrcIP:    srcIP,
			DstIP:    dstIP,
			SrcName:  sourcePodName,
			DstName:  destPodName,
		}
		log.Printf("%+v\n", myLog)
	}

	ret := logs.ExportLogsServiceResponse{PartialSuccess: nil}
	return &ret, nil
}

// loadEnvVars loads environment variables and stores them as global variable
func loadEnvVars() (config, error) {
	cfg := config{}
	var err error

	// load listen address and check if valid
	cfg.listenAddr = os.Getenv("LISTEN_ADDR")
	ip := net.ParseIP(cfg.listenAddr)
	if ip == nil {
		msg := fmt.Sprintf("invalid listen address %s", cfg.listenAddr)
		return cfg, errors.New(msg)
	}
	cfg.listenAddr = ip.String()

	// load listen port and check if valid
	cfg.listenPort, err = strconv.Atoi(os.Getenv("LISTEN_PORT"))
	if err != nil {
		msg := fmt.Sprintf("invalid listen port %s: %v", os.Getenv("LISTEN_PORT"), err)
		return cfg, errors.New(msg)
	}

	return cfg, nil
}

func Initialize() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// Kubernetes 클라이언트 생성
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Error creating Kubernetes client: %v", err)
	}

	// Pod 정보를 가져오기 위한 ListWatch 생성
	listWatcher := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		corev1.NamespaceAll,
		fields.Everything(),
	)

	// 컨트롤러 생성
	_, controller := cache.NewInformer(
		listWatcher,
		&corev1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*corev1.Pod)
				podsMap[pod.Status.PodIP] = pod // Pod 정보를 map에 추가 (IP 주소를 키로 사용)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newPod := newObj.(*corev1.Pod)
				podsMap[newPod.Status.PodIP] = newPod
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*corev1.Pod)
				delete(podsMap, pod.Status.PodIP) // 삭제된 Pod 정보를 map에서 삭제 (IP 주소를 키로 사용)
			},
		},
	)
	// 이벤트 처리를 위한 Goroutine 시작
	stopCh := make(chan struct{}) // stopCh 채널 생성
	//defer close(stopCh)           // 함수 종료 전에 stopCh 채널 닫기

	// 이벤트 처리를 위한 Goroutine 시작
	go func() {
		controller.Run(stopCh)
	}()

	// 클러스터 정보를 가져오는 시간을 확보하기 위한 대기 시간
	time.Sleep(time.Second)

	return nil
}

// main is the entrypoint of this program
func main() {
	// load environment variables
	cfg, err := loadEnvVars()
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	// 클러스터 정보 가져오는 Initialize 함수 실행
	errInit := Initialize()
	if err != nil {
		fmt.Println("Error initializing:", errInit)
		os.Exit(1)
	}

	// Start listening 4317
	addr := fmt.Sprintf("%s:%d", cfg.listenAddr, cfg.listenPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Could not start up server at %s: %v", lis, err)
	}

	// Create a new gRPC server
	grpcServer := grpc.NewServer()

	// Register log collector server
	logServer := LogServer{}
	logs.RegisterLogsServiceServer(grpcServer, logServer)

	// Start serving gRPC requests
	log.Printf("Starting to serve on %s", addr)

	// Start listening gRPC Server for OTEL but with debugging
	log.Printf("Starting to serve...")
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatal("Could not start up gRPC server: ", err)
	}

	// 시그널 처리
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh // 프로그램을 종료할 때까지 대기
}
