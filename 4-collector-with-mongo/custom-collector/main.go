package main

import (
	"context"
	"custom-collector/common"
	"custom-collector/db"
	"strconv"

	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
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
var servicesMap = make(map[string]*corev1.Service)
var id string
var num int

// LogServer is for exporting log handler
type LogServer struct {
	logs.UnimplementedLogsServiceServer
}

// Export interface function for LogServiceServer
func (ls LogServer) Export(c context.Context, request *logs.ExportLogsServiceRequest) (*logs.ExportLogsServiceResponse, error) {
	var logText string
	var index int

	logText = request.String()
	logText = strings.ReplaceAll(logText, `\"`, "")
	logText = strings.ReplaceAll(logText, `}`, "")
	parts := strings.Split(logText, "\\n\"")

	for _, part := range parts[0:] {
		if len(part) <= 0 {
			continue
		}
		index = strings.Index(part, "string_value:\"")
		if index == -1 {
			continue
		}
		result := part[index+len("string_value:\""):]
		words := strings.Fields(result)

		method := words[1]
		path := words[2]
		protocolName := words[3]
		timeStamp := words[0]
		resCode, _ := strconv.ParseInt(words[4], 10, 64)

		srcInform := words[21]
		dstInform := words[20]

		var srcIP string
		var dstIP string
		var srcPort string
		var dstPort string

		var colonIndex int
		// Extract the left and right words based on the colon delimiter (ADDR:PORT)
		colonIndex = strings.LastIndex(srcInform, ":")
		if colonIndex > 0 && colonIndex < len(srcInform)-1 {
			srcIP = strings.TrimSpace(srcInform[:colonIndex])
			srcPort = strings.TrimSpace(srcInform[colonIndex+1:])
		}
		colonIndex = strings.LastIndex(dstInform, ":")
		if colonIndex > 0 && colonIndex < len(dstInform)-1 {
			dstIP = strings.TrimSpace(dstInform[:colonIndex])
			dstPort = strings.TrimSpace(dstInform[colonIndex+1:])
		}

		var srcName string
		var srcNs string
		var srcLabel map[string]string
		var dstName string
		var dstNs string
		var dstLabel map[string]string

		// Get source IP addr, port
		if pod, ok := podsMap[srcIP]; ok {
			srcName = pod.Name
			srcNs = pod.Namespace
			srcLabel = pod.Labels
		} else if service, ok := servicesMap[srcIP]; ok {
			srcName = service.Name
			srcNs = service.Namespace
			srcLabel = service.Labels
		} else {
			srcName = "Not found"
			srcNs = "Not found"
		}

		// Get destination IP addr, port
		if pod, ok := podsMap[dstIP]; ok {
			dstName = pod.Name
			dstNs = pod.Namespace
			dstLabel = pod.Labels
		} else if service, ok := servicesMap[dstIP]; ok {
			dstName = service.Name
			dstNs = service.Namespace
			dstLabel = service.Labels
		} else {
			dstName = "Not found"
			dstNs = "Not found"
		}

		// Create a result log struct - MyLog
		myLog := common.MyLog{
			TimeStamp:        timeStamp,
			Id:               id + strconv.Itoa(num),
			SrcNamespaceName: srcNs,
			SrcPodName:       srcName,
			SrcLabel:		  srcLabel,
			SrcIP:            srcIP,
			SrcPort:          srcPort,
			DstNamespaceName: dstNs,
			DstPodName:       dstName,
			DstLabel:		  dstLabel,
			DstIP:            dstIP,
			DstPort:          dstPort,
			Protocol:         protocolName,
			Method:           method,
			Path:             path,
			ResponseCode:     resCode,
		}

		num++

		// Ignore non HTTP protocols
		if strings.Contains(myLog.Protocol, "HTTP") {
			log.Printf("%+v\n", myLog)
		}

		// Insert data into DB
		err := db.Manager.InsertData(myLog)
		if err != nil {
			log.Printf("[Warn] Unable to insert data into DB: %v", err)
		}

		// 서버로부터 받은 응답 처리
		fmt.Printf("Response from server: %+v\n", response)
	}

	ret := logs.ExportLogsServiceResponse{PartialSuccess: nil}
	return &ret, nil
}

func Initialize() error {
	id = time.Now().Format("20060102150405")
	num = 0

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// Create a Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Error creating Kubernetes client: %v", err)
	}

	// Create a ListWatch to obtain information about pods
	podListWatcher := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		corev1.NamespaceAll,
		fields.Everything(),
	)

	// Create a ListWatch to obtain information about services
	serviceListWatcher := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"services",
		corev1.NamespaceAll,
		fields.Everything(),
	)

	// Create pod controller
	_, podController := cache.NewInformer(
		podListWatcher,
		&corev1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { // Add pod information
				pod := obj.(*corev1.Pod)
				podsMap[pod.Status.PodIP] = pod
			},
			UpdateFunc: func(oldObj, newObj interface{}) { // Update pod information
				newPod := newObj.(*corev1.Pod)
				podsMap[newPod.Status.PodIP] = newPod
			},
			DeleteFunc: func(obj interface{}) { // Remove deleted pod information
				pod := obj.(*corev1.Pod)
				delete(podsMap, pod.Status.PodIP)
			},
		},
	)

	// Create service controller
	_, serviceController := cache.NewInformer(
		serviceListWatcher,
		&corev1.Service{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { // Add service information
				service := obj.(*corev1.Service)
				servicesMap[service.Spec.ClusterIP] = service
			},
			UpdateFunc: func(oldObj, newObj interface{}) { // Update service information
				newService := newObj.(*corev1.Service)
				servicesMap[newService.Spec.ClusterIP] = newService
			},
			DeleteFunc: func(obj interface{}) {
				service := obj.(*corev1.Service)
				delete(servicesMap, service.Spec.ClusterIP) // Remove deleted service information
			},
		},
	)

	// Create stopCh channel
	stopCh := make(chan struct{})

	// Start a Goroutine for processing pod events
	go func() {
		podController.Run(stopCh)
	}()

	//Start a Goroutine for processing service events
	go func() {
		serviceController.Run(stopCh)
	}()

	// Wait time to ensure obtaining cluster information
	time.Sleep(time.Second)

	return nil
}

// main is the entrypoint of this program
func main() {
	// Init DB
	_, err := db.New()
	if err != nil {
		log.Fatalf("Unable to intialize DB: %v", err)
	}

	// load environment variables
	cfg, err := common.LoadEnvVars()
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	// Get cluster information and create controllers for pods and services
	errInit := Initialize()
	if err != nil {
		fmt.Println("Error initializing:", errInit)
		os.Exit(1)
	}

	// Start listening 4317
	addr := fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ListenPort)
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

	// Signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh
}
