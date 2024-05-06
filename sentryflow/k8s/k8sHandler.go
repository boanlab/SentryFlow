// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/5gsec/SentryFlow/config"
	"github.com/5gsec/SentryFlow/types"

	"gopkg.in/yaml.v2"
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
		log.Fatal("[InitK8sClient] Failed to initialize Kubernetes client")
		return false
	}

	// Initialize Kubernetes clientSet
	K8sH.clientSet, err = kubernetes.NewForConfig(K8sH.config)
	if err != nil {
		log.Fatal("[InitK8sClient] Failed to initialize Kubernetes client")
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

	log.Printf("[InitK8sClient] Initialized Kubernetes client")

	return true
}

// initExistingResources Function that creates a mapping table for existing pods and services to IPs
// This is required since informers are NOT going to see existing resources until they are updated, created or deleted
// @todo: Refactor this function, this is kind of messy
func (k8s *KubernetesHandler) initExistingResources() {
	// List existing Pods
	podList, err := k8s.clientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Print("[K8s] Error listing Pods:", err.Error())
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
		log.Print("[K8s] Error listing Services:", err.Error())
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

	log.Printf("[RunInformers] Started all Kubernetes informers")
}

// == //

// PatchIstioConfigMap Function that patches the Istio's configmap for meshConfig
// This will make istio know that there is an exporter with envoyOtelAls
func PatchIstioConfigMap() bool {
	var meshConfig map[string]interface{}

	// Get the ConfigMap istio-system/istio
	configMap, err := K8sH.clientSet.CoreV1().ConfigMaps("istio-system").Get(context.Background(), "istio", v1.GetOptions{})
	if err != nil {
		log.Fatalf("[PatchIstioConfigMap] Unable to retrieve ConfigMap istio-system/istio :%v", err)
		return false
	}

	// Unmarshal the YAML string into meshConfig
	if err = yaml.Unmarshal([]byte(configMap.Data["mesh"]), &meshConfig); err != nil {
		log.Fatalf("[PatchIstioConfigMap] Unable to unmarshall ConfigMap istio-system/istio :%v", err)
		return false
	}

	if _, evyAccLogExist := meshConfig["enableEnvoyAccessLogService"]; evyAccLogExist {
		log.Printf("[PatchIstioConfigMap] Overwrite the contents of \"enableEnvoyAccessLogService\"")
	}
	meshConfig["enableEnvoyAccessLogService"] = true

	if _, evyAccLogExist := meshConfig["defaultConfig"].(map[interface{}]interface{})["envoyAccessLogService"]; evyAccLogExist {
		log.Printf("[PatchIstioConfigMap] Overwrite the contents of \"defaultConfig.envoyAccessLogService\"")
	}
	meshConfig["defaultConfig"].(map[interface{}]interface{})["envoyAccessLogService"] = map[string]string{
		"address": "sentryflow.sentryflow.svc.cluster.local:4317",
	}

	if _, evyMetricsExist := meshConfig["defaultConfig"].(map[interface{}]interface{})["envoyMetricsService"]; evyMetricsExist {
		log.Printf("[PatchIstioConfigMap] Overwrite the contents of \"defaultConfig.envoyMetricsService\"")
	}
	meshConfig["defaultConfig"].(map[interface{}]interface{})["envoyMetricsService"] = map[string]string{
		"address": "sentryflow.sentryflow.svc.cluster.local:4317",
	}

	// Update defaultProviders.accessLogs
	if defProviders, exists := meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"]; exists {
		newDefProviders := defProviders.([]interface{})

		exists = false
		for _, entry := range newDefProviders {
			if entry == "sentryflow" { // If "sentryflow" already exists
				log.Printf("[PatchIstioConfigMap] istio-system/istio ConfigMap has SentryFlow under defaultProviders.accessLogs, ignoring...")
				exists = true
				break
			}
		}

		if !exists { // If "sentryflow" does not exist
			newDefProviders = append(newDefProviders, "sentryflow")
			meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"] = newDefProviders
		}
	} else { // If it does not exist
		meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"] = []string{"sentryflow"}
	}

	// ExtensionProvider for our service
	extensionProvider := map[interface{}]interface{}{
		"name": "sentryflow",
		"envoyOtelAls": map[interface{}]interface{}{
			"service": "sentryflow.sentryflow.svc.cluster.local",
			"port":    config.GlobalConfig.CollectorPort,
		},
	}

	// Update extensionProviders
	if extensionProviders, exists := meshConfig["extensionProviders"]; exists {
		newExtensionProviders, ok := extensionProviders.([]interface{})
		if !ok {
			log.Printf("[PatchIstioConfigMap] 'extensionProviders' in istio-system/istio ConfigMap has an unexpected type")
		}

		exists = false
		for _, entry := range newExtensionProviders {
			if entryMap, ok := entry.(map[interface{}]interface{}); !ok {
				log.Printf("[PatchIstioConfigMap] 'extensionProviders' in istio-system/istio ConfigMap has an unexpected type")
			} else if entryMap["name"] == "sentryflow" { // If "sentryflow" already exists
				log.Printf("[PatchIstioConfigMap] istio-system/istio ConfigMap has sentryflow under extensionProviders, ignoring... ")
				exists = true
				break
			}
		}

		if !exists {
			meshConfig["extensionProviders"] = append(extensionProviders.([]map[interface{}]interface{}), extensionProvider)
		}
	} else { // If it does not exist
		meshConfig["extensionProviders"] = []map[interface{}]interface{}{extensionProvider}
	}

	// Update the ConfigMap data with the modified meshConfig
	updatedMeshConfig, err := yaml.Marshal(meshConfig)
	if err != nil {
		log.Fatalf("[PatchIstioConfigMap] Unable to marshal updated meshConfig to YAML: %v", err)
		return false
	}

	// Convert the []byte to string
	configMap.Data["mesh"] = string(updatedMeshConfig)

	// Preview changes, for debugging
	if config.GlobalConfig.Debug {
		log.Printf("[PatchIstioConfigMap] Patching istio-system/istio ConfigMap as: \n%v", configMap)
	}

	// Patch the ConfigMap
	if updatedConfigMap, err := K8sH.clientSet.CoreV1().ConfigMaps("istio-system").Update(context.Background(), configMap, v1.UpdateOptions{}); err != nil {
		log.Fatalf("[PatchIstioConfigMap] Unable to update configmap istio-system/istio :%v", err)
	} else {
		log.Printf("[PatchIstioConfigMap] Updated istio-system/istio ConfigMap")

		if config.GlobalConfig.Debug {
			log.Printf("%v", updatedConfigMap)
		}
	}

	log.Printf("[PatchIstioConfigMap] Patched Istio ConfigMap")

	return true
}

// UnpatchIstioConfigMap Function
func UnpatchIstioConfigMap() bool {
	// @todo: Remove SentryFlow collector from Kubernetes
	return true
}

// == //

// PatchNamespaces Function that patches namespaces for adding 'istio-injection'
func PatchNamespaces() bool {
	namespaces, err := K8sH.clientSet.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("[PatchNamespaces] Unable to list namespaces: %v", err)
		return false
	}

	for _, namespace := range namespaces.Items {
		// Skip the following namespaces
		if namespace.Name == "sentryflow" {
			continue
		}

		namespace.Labels["istio-injection"] = "enabled"

		// Patch the namespace
		if _, err := K8sH.clientSet.CoreV1().Namespaces().Update(context.TODO(), &namespace, v1.UpdateOptions{FieldManager: "patcher"}); err != nil {
			log.Printf("[PatchNamespaces] Unable to update namespace %s: %v", namespace.Name, err)
			return false
		}

		log.Printf("[PatchNamespaces] Updated Namespace: %s\n", namespace.Name)
	}

	log.Printf("[PatchNamespaces] Updated all namespaces")

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
		log.Fatalf("[PatchDeployments] Unable to list deployments: %v", err)
		return false
	}

	for _, deployment := range deployments.Items {
		// Skip the following namespaces
		if deployment.Namespace == "sentryflow" {
			continue
		}

		// Restart the deployment
		if err := K8sH.restartDeployment(deployment.Namespace, deployment.Name); err != nil {
			log.Fatalf("[PatchDeployments] Unable to restart deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
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