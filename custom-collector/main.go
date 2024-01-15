package main

import (
	"custom-collector/common"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var podsMap = make(map[string]*corev1.Pod)
var servicesMap = make(map[string]*corev1.Service)
var id string
var num int

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

	// Start serving gRPC requests
	log.Printf("Starting to serve on %s", addr)

	// Signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh
}
