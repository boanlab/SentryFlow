package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"log"
	"os"
	"time"
)

// Handler represents the k8s-api handler
type Handler struct {
	config    *rest.Config
	clientSet *kubernetes.Clientset

	listWatchers map[string]*cache.ListWatch
	informers    map[string]cache.Controller
	podMap       map[string]*corev1.Pod     // This map is NOT thread safe, meaning that race condition might occur
	svcMap       map[string]*corev1.Service // This map is NOT thread safe, meaning that race condition might occur
}

// Manager is for global reference
var Manager *Handler

// NewHandler creates a new Kubernetes handler object
func NewHandler() (*Handler, error) {
	h := &Handler{
		listWatchers: make(map[string]*cache.ListWatch),
		podMap:       make(map[string]*corev1.Pod),
		svcMap:       make(map[string]*corev1.Service),
		informers:    make(map[string]cache.Controller),
	}
	var err error

	// Initialize in cluster config
	h.config, err = rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Initialize Kubernetes clientSet
	h.clientSet, err = kubernetes.NewForConfig(h.config)
	if err != nil {
		return nil, err
	}

	watchTargets := []string{"pods", "services"}

	// Initialize watchers and informers for services and pods
	// This will not run the informers yet
	h.initWatchers(watchTargets)
	h.initInformers()

	// Everything was successful
	Manager = h
	return h, nil
}

// initWatchers initializes watchers for pods and services in cluster
func (h *Handler) initWatchers(watchTargets []string) {
	//  Initialize watch for pods and services
	for _, target := range watchTargets {
		watcher := cache.NewListWatchFromClient(
			h.clientSet.CoreV1().RESTClient(),
			target,
			corev1.NamespaceAll,
			fields.Everything(),
		)
		h.listWatchers[target] = watcher
	}
}

// initInformers initializes informers for services in
func (h *Handler) initInformers() {
	// Create Pod controller informer
	_, pc := cache.NewInformer(
		h.listWatchers["pods"],
		&corev1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { // Add pod information
				pod := obj.(*corev1.Pod)
				h.podMap[pod.Status.PodIP] = pod
			},
			UpdateFunc: func(oldObj, newObj interface{}) { // Update pod information
				newPod := newObj.(*corev1.Pod)
				h.podMap[newPod.Status.PodIP] = newPod
			},
			DeleteFunc: func(obj interface{}) { // Remove deleted pod information
				pod := obj.(*corev1.Pod)
				delete(h.podMap, pod.Status.PodIP)
			},
		},
	)

	h.informers["pods"] = pc

	// Create Service controller informer
	_, sc := cache.NewInformer(
		h.listWatchers["services"],
		&corev1.Service{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { // Add service information
				service := obj.(*corev1.Service)
				h.svcMap[service.Spec.ClusterIP] = service
			},
			UpdateFunc: func(oldObj, newObj interface{}) { // Update service information
				newService := newObj.(*corev1.Service)
				h.svcMap[newService.Spec.ClusterIP] = newService
			},
			DeleteFunc: func(obj interface{}) {
				service := obj.(*corev1.Service)
				delete(h.svcMap, service.Spec.ClusterIP) // Remove deleted service information
			},
		},
	)

	h.informers["services"] = sc
}

// RunInformers starts running informers
func (h *Handler) RunInformers(stopCh chan os.Signal) {
	for name, informer := range h.informers {
		name := name
		informer := informer
		go func() {
			log.Printf("[K8s] Started informers for %s", name)
			tmpCh := make(chan struct{}) // Need to convert chan os.Signal into chan struct{}
			go func() {
				<-stopCh
				close(tmpCh)
			}()
			informer.Run(tmpCh)
		}()
	}

	log.Printf("[K8s] Started all informers")
}

// IPtoResource returns the pointer address of the Kubernetes resource with the given IP
// This will look for the pod first and then the services, will return nil if no such IP found
// Since this is not thread safe, perform retries for this function since there might be informers
// updating the map for service and pods as well
func (h *Handler) IPtoResource(ipAddr string) interface{} {
	// Look for pod map first
	pod, ok := h.podMap[ipAddr]
	if ok {
		return pod
	}

	// Look for service map
	service, ok := h.svcMap[ipAddr]
	if ok {
		return service
	}

	return nil
}
