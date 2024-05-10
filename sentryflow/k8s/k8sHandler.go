// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/json"

	"github.com/5gsec/SentryFlow/types"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// == //

// K8sH global reference for Kubernetes Handler
var K8sH *KubernetesHandler

// init Function
func init() {
	K8sH = NewK8sHandler()
}

// KubernetesHandler Structure
type KubernetesHandler struct {
	config    *rest.Config
	clientSet *kubernetes.Clientset

	watchers  map[string]*cache.ListWatch
	informers map[string]cache.Controller

	podMap     map[string]*corev1.Pod     // NOT thread safe
	serviceMap map[string]*corev1.Service // NOT thread safe
}

// NewK8sHandler Function
func NewK8sHandler() *KubernetesHandler {
	kh := &KubernetesHandler{
		watchers:  make(map[string]*cache.ListWatch),
		informers: make(map[string]cache.Controller),

		podMap:     make(map[string]*corev1.Pod),
		serviceMap: make(map[string]*corev1.Service),
	}

	return kh
}

// == //

// InitK8sClient Function
func InitK8sClient() bool {
	var err error

	// Initialize in cluster config
	K8sH.config, err = rest.InClusterConfig()
	if err != nil {
		log.Print("[InitK8sClient] Failed to initialize Kubernetes client")
		return false
	}

	// Initialize Kubernetes clientSet
	K8sH.clientSet, err = kubernetes.NewForConfig(K8sH.config)
	if err != nil {
		log.Print("[InitK8sClient] Failed to initialize Kubernetes client")
		return false
	}

	// Create a mapping table for existing pods and services to IPs
	K8sH.initExistingResources()

	watchTargets := []string{"pods", "services"}

	//  Initialize watchers for pods and services
	for _, target := range watchTargets {
		watcher := cache.NewListWatchFromClient(
			K8sH.clientSet.CoreV1().RESTClient(),
			target,
			corev1.NamespaceAll,
			fields.Everything(),
		)
		K8sH.watchers[target] = watcher
	}

	// Initialize informers
	K8sH.initInformers()

	log.Print("[InitK8sClient] Initialized Kubernetes client")

	return true
}

// initExistingResources Function that creates a mapping table for existing pods and services to IPs
// This is required since informers are NOT going to see existing resources until they are updated, created or deleted
// @todo: Refactor this function, this is kind of messy
func (k8s *KubernetesHandler) initExistingResources() {
	// List existing Pods
	podList, err := k8s.clientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("[K8s] Failed to get Pods: %v", err.Error())
	}

	// Add existing Pods to the podMap
	for _, pod := range podList.Items {
		currentPod := pod
		k8s.podMap[pod.Status.PodIP] = &currentPod
		log.Printf("[K8s] Add existing pod %s: %s/%s", pod.Status.PodIP, pod.Namespace, pod.Name)
	}

	// List existing Services
	serviceList, err := k8s.clientSet.CoreV1().Services(corev1.NamespaceAll).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("[K8s] Failed to get Services: %v", err.Error())
	}

	// Add existing Services to the serviceMap
	for _, service := range serviceList.Items {
		currentService := service

		// Check if the service has a LoadBalancer type
		if service.Spec.Type == "LoadBalancer" {
			for _, lbIngress := range service.Status.LoadBalancer.Ingress {
				lbIP := lbIngress.IP
				if lbIP != "" {
					k8s.serviceMap[lbIP] = &currentService
					log.Printf("[K8s] Add existing service (LoadBalancer) %s: %s/%s", lbIP, service.Namespace, service.Name)
				}
			}
		} else {
			k8s.serviceMap[service.Spec.ClusterIP] = &currentService
			if len(service.Spec.ExternalIPs) != 0 {
				for _, eIP := range service.Spec.ExternalIPs {
					k8s.serviceMap[eIP] = &currentService
					log.Printf("[K8s] Add existing service %s: %s/%s", eIP, service.Namespace, service.Name)
				}
			}
		}
	}
}

// initInformers Function that initializes informers for services and pods in a cluster
func (k8s *KubernetesHandler) initInformers() {
	// Create Pod controller informer
	_, pc := cache.NewInformer(
		k8s.watchers["pods"],
		&corev1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { // Add pod
				pod := obj.(*corev1.Pod)
				k8s.podMap[pod.Status.PodIP] = pod
			},
			UpdateFunc: func(oldObj, newObj interface{}) { // Update pod
				newPod := newObj.(*corev1.Pod)
				k8s.podMap[newPod.Status.PodIP] = newPod
			},
			DeleteFunc: func(obj interface{}) { // Remove deleted pod
				pod := obj.(*corev1.Pod)
				delete(k8s.podMap, pod.Status.PodIP)
			},
		},
	)
	k8s.informers["pods"] = pc

	// Create Service controller informer
	_, sc := cache.NewInformer(
		k8s.watchers["services"],
		&corev1.Service{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { // Add service
				service := obj.(*corev1.Service)

				if service.Spec.Type == "LoadBalancer" {
					for _, lbIngress := range service.Status.LoadBalancer.Ingress {
						lbIP := lbIngress.IP
						if lbIP != "" {
							k8s.serviceMap[lbIP] = service
						}
					}
				} else {
					k8s.serviceMap[service.Spec.ClusterIP] = service
					if len(service.Spec.ExternalIPs) != 0 {
						for _, eIP := range service.Spec.ExternalIPs {
							k8s.serviceMap[eIP] = service
						}
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) { // Update service
				newService := newObj.(*corev1.Service)
				if newService.Spec.Type == "LoadBalancer" {
					for _, lbIngress := range newService.Status.LoadBalancer.Ingress {
						lbIP := lbIngress.IP
						if lbIP != "" {
							k8s.serviceMap[lbIP] = newService
						}
					}
				} else {
					k8s.serviceMap[newService.Spec.ClusterIP] = newService
					if len(newService.Spec.ExternalIPs) != 0 {
						for _, eIP := range newService.Spec.ExternalIPs {
							k8s.serviceMap[eIP] = newService
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
							delete(k8s.serviceMap, lbIP)
						}
					}
				} else {
					delete(k8s.serviceMap, service.Spec.ClusterIP) // Remove deleted service
					if len(service.Spec.ExternalIPs) != 0 {
						for _, eIP := range service.Spec.ExternalIPs {
							delete(k8s.serviceMap, eIP)
						}
					}
				}
			},
		},
	)
	k8s.informers["services"] = sc
}

// == //

// RunInformers Function that starts running informers
func RunInformers(stopChan chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)

	for name, informer := range K8sH.informers {
		name := name
		informer := informer
		go func() {
			log.Printf("[RunInformers] Starting an informer for %s", name)
			informer.Run(stopChan)
			defer wg.Done()
		}()
	}

	log.Print("[RunInformers] Started all Kubernetes informers")
}

// getConfigMap Function
func (k8s *KubernetesHandler) getConfigMap(namespace, name string) (string, error) {
	cm, err := k8s.clientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		log.Printf("[K8s] Failed to get ConfigMaps: %v", err)
		return "", err
	}

	// convert data to string
	data, err := json.Marshal(cm.Data)
	if err != nil {
		log.Printf("[K8s] Failed to marshal ConfigMap: %v", err)
		return "", err
	}

	return string(data), nil
}

// updateConfigMap Function
func (k8s *KubernetesHandler) updateConfigMap(namespace, name, data string) error {
	cm, err := k8s.clientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		log.Printf("[K8s] Failed to get ConfigMap: %v", err)
		return err
	}

	if _, ok := cm.Data["mesh"]; !ok {
		return errors.New("[K8s] Unable to find field \"mesh\" from Istio config")
	}

	cm.Data["mesh"] = data
	if _, err := k8s.clientSet.CoreV1().ConfigMaps(namespace).Update(context.Background(), cm, v1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// PatchNamespaces Function that patches namespaces for adding 'istio-injection'
func PatchNamespaces() bool {
	namespaces, err := K8sH.clientSet.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Printf("[PatchNamespaces] Failed to get Namespaces: %v", err)
		return false
	}

	for _, ns := range namespaces.Items {
		namespace := ns.DeepCopy()

		// Skip the following namespaces
		if namespace.Name == "sentryflow" {
			continue
		}

		namespace.Labels["istio-injection"] = "enabled"

		// Patch the namespace
		if _, err := K8sH.clientSet.CoreV1().Namespaces().Update(context.TODO(), namespace, v1.UpdateOptions{FieldManager: "patcher"}); err != nil {
			log.Printf("[PatchNamespaces] Failed to update Namespace %s: %v", namespace.Name, err)
			return false
		}

		log.Printf("[PatchNamespaces] Updated Namespace %s", namespace.Name)
	}

	log.Print("[PatchNamespaces] Updated all Namespaces")

	return true
}

// restartDeployment Function that performs a rolling restart for a deployment in the specified namespace
// @todo: fix this, this DOES NOT restart deployments
func (k8s *KubernetesHandler) restartDeployment(namespace string, deploymentName string) error {
	deploymentClient := k8s.clientSet.AppsV1().Deployments(namespace)

	// Get the deployment to retrieve the current spec
	deployment, err := deploymentClient.Get(context.Background(), deploymentName, v1.GetOptions{})
	if err != nil {
		return err
	}

	// Trigger a rolling restart by updating the deployment's labels or annotations
	deployment.Spec.Template.ObjectMeta.Labels["restartedAt"] = v1.Now().String()

	// Update the deployment to trigger the rolling restart
	_, err = deploymentClient.Update(context.TODO(), deployment, v1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// RestartDeployments Function that restarts the deployments in the namespaces with "istio-injection=enabled"
func RestartDeployments() bool {
	deployments, err := K8sH.clientSet.AppsV1().Deployments("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Printf("[PatchDeployments] Failed to get Deployments: %v", err)
		return false
	}

	for _, deployment := range deployments.Items {
		// Skip the following namespaces
		if deployment.Namespace == "sentryflow" {
			continue
		}

		// Restart the deployment
		if err := K8sH.restartDeployment(deployment.Namespace, deployment.Name); err != nil {
			log.Printf("[PatchDeployments] Failed to restart Deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
			return false
		}

		log.Printf("[PatchDeployments] Deployment %s/%s restarted", deployment.Namespace, deployment.Name)
	}

	log.Print("[PatchDeployments] Restarted all patched deployments")

	return true
}

// == //

// lookupIPAddress Function
func lookupIPAddress(ipAddr string) interface{} {
	// Look for pod map
	pod, ok := K8sH.podMap[ipAddr]
	if ok {
		return pod
	}

	// Look for service map
	service, ok := K8sH.serviceMap[ipAddr]
	if ok {
		return service
	}

	return nil
}

// LookupK8sResource Function
func LookupK8sResource(srcIP string) types.K8sResource {
	ret := types.K8sResource{
		Namespace: "Unknown",
		Name:      "Unknown",
		Labels:    make(map[string]string),
		Type:      types.K8sResourceTypeUnknown,
	}

	// Find Kubernetes resource from source IP (service or a pod)
	raw := lookupIPAddress(srcIP)

	// Currently supports Service or Pod
	switch raw.(type) {
	case *corev1.Pod:
		pod, ok := raw.(*corev1.Pod)
		if ok {
			ret.Namespace = pod.Namespace
			ret.Name = pod.Name
			ret.Labels = pod.Labels
			ret.Type = types.K8sResourceTypePod
		}
	case *corev1.Service:
		svc, ok := raw.(*corev1.Service)
		if ok {
			ret.Namespace = svc.Namespace
			ret.Name = svc.Name
			ret.Labels = svc.Labels
			ret.Type = types.K8sResourceTypeService
		}
	default:
		ret.Type = types.K8sResourceTypeUnknown
	}

	return ret
}

// == //
