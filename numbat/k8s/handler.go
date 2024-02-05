package k8s

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Look for existing resources in the cluster, create map
	h.initExistingResources()

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

// initExistingResources will create a mapping table for existing services and pods into IPs
// This is required since informers are NOT going to see existing resources until they are updated, created or deleted
func (h *Handler) initExistingResources() {
	// List existing Pods
	podList, err := h.clientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatal("Error listing Pods:", err.Error())
	}

	// Add existing Pods to the podMap
	for _, pod := range podList.Items {
		h.podMap[pod.Status.PodIP] = &pod
		log.Printf("[K8s] Add existing pod %s: %s/%s", pod.Status.PodIP, pod.Namespace, pod.Name)
	}

	// List existing Services
	serviceList, err := h.clientSet.CoreV1().Services(corev1.NamespaceAll).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatal("Error listing Services:", err.Error())
	}

	// Add existing Services to the svcMap
	for _, service := range serviceList.Items {
		// Check if the service has a LoadBalancer type
		if service.Spec.Type == "LoadBalancer" {
			for _, lbIngress := range service.Status.LoadBalancer.Ingress {
				lbIP := lbIngress.IP
				if lbIP != "" {
					h.svcMap[lbIP] = &service
					log.Printf("[K8s] Add existing service (LoadBalancer) %s: %s/%s", lbIP, service.Namespace, service.Name)
				}
			}
		} else {
			h.svcMap[service.Spec.ClusterIP] = &service
			if len(service.Spec.ExternalIPs) != 0 {
				for _, eIP := range service.Spec.ExternalIPs {
					h.svcMap[eIP] = &service
					log.Printf("[K8s] Add existing service %s: %s/%s", eIP, service.Namespace, service.Name)
				}
			}
		}
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

				if service.Spec.Type == "LoadBalancer" {
					for _, lbIngress := range service.Status.LoadBalancer.Ingress {
						lbIP := lbIngress.IP
						if lbIP != "" {
							h.svcMap[lbIP] = service
						}
					}
				} else {
					h.svcMap[service.Spec.ClusterIP] = service
					if len(service.Spec.ExternalIPs) != 0 {
						for _, eIP := range service.Spec.ExternalIPs {
							h.svcMap[eIP] = service
						}
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) { // Update service information
				newService := newObj.(*corev1.Service)
				if newService.Spec.Type == "LoadBalancer" {
					for _, lbIngress := range newService.Status.LoadBalancer.Ingress {
						lbIP := lbIngress.IP
						if lbIP != "" {
							h.svcMap[lbIP] = newService
						}
					}
				} else {
					h.svcMap[newService.Spec.ClusterIP] = newService
					if len(newService.Spec.ExternalIPs) != 0 {
						for _, eIP := range newService.Spec.ExternalIPs {
							h.svcMap[eIP] = newService
						}
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				service := obj.(*corev1.Service)
				if service.Spec.Type == "LoadBalancer" {
					for _, lbIngress := range service.Status.LoadBalancer.Ingress {
						lbIP := lbIngress.IP
						if lbIP != "" {
							delete(h.svcMap, lbIP)
						}
					}
				} else {
					delete(h.svcMap, service.Spec.ClusterIP) // Remove deleted service information
					if len(service.Spec.ExternalIPs) != 0 {
						for _, eIP := range service.Spec.ExternalIPs {
							delete(h.svcMap, eIP)
						}
					}
				}
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
