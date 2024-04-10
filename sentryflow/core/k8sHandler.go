// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/5GSEC/SentryFlow/config"
	"github.com/5GSEC/SentryFlow/types"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// K8s global reference for Kubernetes Handler
var K8s *K8sHandler

// init Function
func init() {
	K8s = NewK8sHandler()
}

// K8sHandler Structure
type K8sHandler struct {
	config    *rest.Config
	clientSet *kubernetes.Clientset

	listWatchers map[string]*cache.ListWatch
	informers    map[string]cache.Controller
	podMap       map[string]*corev1.Pod     // This map is NOT thread safe, meaning that race condition might occur
	svcMap       map[string]*corev1.Service // This map is NOT thread safe, meaning that race condition might occur
}

// NewK8sHandler Function
func NewK8sHandler() *K8sHandler {
	kh := &K8sHandler{
		listWatchers: make(map[string]*cache.ListWatch),
		podMap:       make(map[string]*corev1.Pod),
		svcMap:       make(map[string]*corev1.Service),
		informers:    make(map[string]cache.Controller),
	}

	return kh
}

// InitK8sClient Function
func (kh *K8sHandler) InitK8sClient() bool {
	var err error

	// Initialize in cluster config
	kh.config, err = rest.InClusterConfig()
	if err != nil {
		return false
	}

	// Initialize Kubernetes clientSet
	kh.clientSet, err = kubernetes.NewForConfig(kh.config)
	if err != nil {
		return false
	}

	watchTargets := []string{"pods", "services"}

	// Look for existing resources in the cluster, create map
	kh.initExistingResources()

	// Initialize watchers and informers for services and pods
	// This will not run the informers yet
	kh.initWatchers(watchTargets)
	kh.initInformers()

	return true
}

// initWatchers initializes watchers for pods and services in cluster
func (kh *K8sHandler) initWatchers(watchTargets []string) {
	//  Initialize watch for pods and services
	for _, target := range watchTargets {
		watcher := cache.NewListWatchFromClient(
			kh.clientSet.CoreV1().RESTClient(),
			target,
			corev1.NamespaceAll,
			fields.Everything(),
		)
		kh.listWatchers[target] = watcher
	}
}

// initExistingResources will create a mapping table for existing services and pods into IPs
// This is required since informers are NOT going to see existing resources until they are updated, created or deleted
// Todo: Refactor this function, this is kind of messy
func (kh *K8sHandler) initExistingResources() {
	// List existing Pods
	podList, err := kh.clientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Print("Error listing Pods:", err.Error())
	}

	// Add existing Pods to the podMap
	for _, pod := range podList.Items {
		currentPod := pod
		kh.podMap[pod.Status.PodIP] = &currentPod
		log.Printf("[K8s] Add existing pod %s: %s/%s", pod.Status.PodIP, pod.Namespace, pod.Name)
	}

	// List existing Services
	serviceList, err := kh.clientSet.CoreV1().Services(corev1.NamespaceAll).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Print("Error listing Services:", err.Error())
	}

	// Add existing Services to the svcMap
	for _, service := range serviceList.Items {
		currentService := service // This will solve G601 for gosec

		// Check if the service has a LoadBalancer type
		if service.Spec.Type == "LoadBalancer" {
			for _, lbIngress := range service.Status.LoadBalancer.Ingress {
				lbIP := lbIngress.IP
				if lbIP != "" {
					kh.svcMap[lbIP] = &currentService
					log.Printf("[K8s] Add existing service (LoadBalancer) %s: %s/%s", lbIP, service.Namespace, service.Name)
				}
			}
		} else {
			kh.svcMap[service.Spec.ClusterIP] = &currentService
			if len(service.Spec.ExternalIPs) != 0 {
				for _, eIP := range service.Spec.ExternalIPs {
					kh.svcMap[eIP] = &currentService
					log.Printf("[K8s] Add existing service %s: %s/%s", eIP, service.Namespace, service.Name)
				}
			}
		}
	}
}

// initInformers initializes informers for services and pods in cluster
func (kh *K8sHandler) initInformers() {
	// Create Pod controller informer
	_, pc := cache.NewInformer(
		kh.listWatchers["pods"],
		&corev1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { // Add pod information
				pod := obj.(*corev1.Pod)
				kh.podMap[pod.Status.PodIP] = pod
			},
			UpdateFunc: func(oldObj, newObj interface{}) { // Update pod information
				newPod := newObj.(*corev1.Pod)
				kh.podMap[newPod.Status.PodIP] = newPod
			},
			DeleteFunc: func(obj interface{}) { // Remove deleted pod information
				pod := obj.(*corev1.Pod)
				delete(kh.podMap, pod.Status.PodIP)
			},
		},
	)

	kh.informers["pods"] = pc

	// Create Service controller informer
	_, sc := cache.NewInformer(
		kh.listWatchers["services"],
		&corev1.Service{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) { // Add service information
				service := obj.(*corev1.Service)

				if service.Spec.Type == "LoadBalancer" {
					for _, lbIngress := range service.Status.LoadBalancer.Ingress {
						lbIP := lbIngress.IP
						if lbIP != "" {
							kh.svcMap[lbIP] = service
						}
					}
				} else {
					kh.svcMap[service.Spec.ClusterIP] = service
					if len(service.Spec.ExternalIPs) != 0 {
						for _, eIP := range service.Spec.ExternalIPs {
							kh.svcMap[eIP] = service
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
							kh.svcMap[lbIP] = newService
						}
					}
				} else {
					kh.svcMap[newService.Spec.ClusterIP] = newService
					if len(newService.Spec.ExternalIPs) != 0 {
						for _, eIP := range newService.Spec.ExternalIPs {
							kh.svcMap[eIP] = newService
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
							delete(kh.svcMap, lbIP)
						}
					}
				} else {
					delete(kh.svcMap, service.Spec.ClusterIP) // Remove deleted service information
					if len(service.Spec.ExternalIPs) != 0 {
						for _, eIP := range service.Spec.ExternalIPs {
							delete(kh.svcMap, eIP)
						}
					}
				}
			},
		},
	)

	kh.informers["services"] = sc
}

// RunInformers starts running informers
func (kh *K8sHandler) RunInformers(stopCh chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	for name, informer := range kh.informers {
		name := name
		informer := informer
		go func() {
			log.Printf("[K8s] Started informers for %s", name)
			informer.Run(stopCh)

			defer wg.Done()
		}()
	}

	log.Printf("[K8s] Started all informers")
}

// lookupIPAddress Function
func (kh *K8sHandler) lookupIPAddress(ipAddr string) interface{} {
	// Look for pod map first
	pod, ok := kh.podMap[ipAddr]
	if ok {
		return pod
	}

	// Look for service map
	service, ok := kh.svcMap[ipAddr]
	if ok {
		return service
	}

	return nil
}

// LookupNetworkedResource Function
func LookupNetworkedResource(srcIP string) types.K8sNetworkedResource {
	ret := types.K8sNetworkedResource{
		Name:      "Unknown",
		Namespace: "Unknown",
		Labels:    make(map[string]string),
		Type:      types.K8sResourceTypeUnknown,
	}

	// Find Kubernetes resource from source IP (service or a pod)
	raw := K8s.lookupIPAddress(srcIP)

	// Currently supports Service or Pod
	switch raw.(type) {
	case *corev1.Pod:
		pod, ok := raw.(*corev1.Pod)
		if ok {
			ret.Name = pod.Name
			ret.Namespace = pod.Namespace
			ret.Labels = pod.Labels
			ret.Type = types.K8sResourceTypePod
		}
	case *corev1.Service:
		svc, ok := raw.(*corev1.Service)
		if ok {
			ret.Name = svc.Name
			ret.Namespace = svc.Namespace
			ret.Labels = svc.Labels
			ret.Type = types.K8sResourceTypeService
		}
	default:
		ret.Type = types.K8sResourceTypeUnknown
	}

	return ret
}

// PatchIstioConfigMap patches the Istio's configmap for meshConfig
// This will make istio know that there is an exporter with envoyOtelAls
func (kh *K8sHandler) PatchIstioConfigMap() error {
	// Get the ConfigMap istio-system/istio
	configMap, err := kh.clientSet.CoreV1().
		ConfigMaps("istio-system").
		Get(context.Background(), "istio", v1.GetOptions{})
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to retrieve configmap istio-system/istio :%v", err)
		return err
	}

	// Define a map to represent the structure of the mesh configuration
	var meshConfig map[string]interface{}

	// Unmarshal the YAML string into the map
	meshConfigStr := configMap.Data["mesh"]
	err = yaml.Unmarshal([]byte(meshConfigStr), &meshConfig)
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to unmarshall configmap istio-system/istio :%v", err)
		return err
	}

	_, eeaExist := meshConfig["enableEnvoyAccessLogService"]
	if eeaExist {
		log.Printf("Overwrite the contents of \"enableEnvoyAccessLogService\"")
	}
	meshConfig["enableEnvoyAccessLogService"] = true

	_, ealExist := meshConfig["defaultConfig"].(map[interface{}]interface{})["envoyAccessLogService"]
	if ealExist {
		log.Printf("Overwrite the contents of \"defaultConfig.envoyAccessLogService\"")
	}
	meshConfig["defaultConfig"].(map[interface{}]interface{})["envoyAccessLogService"] = map[string]string{
		"address": "sentryflow.sentryflow.svc.cluster.local:4317",
	}

	_, emExist := meshConfig["defaultConfig"].(map[interface{}]interface{})["envoyMetricsService"]
	if emExist {
		log.Printf("Overwrite the contents of \"defaultConfig.envoyMetricsService\"")
	}
	meshConfig["defaultConfig"].(map[interface{}]interface{})["envoyMetricsService"] = map[string]string{
		"address": "sentryflow.sentryflow.svc.cluster.local:4317",
	}

	// Work with defaultProviders.accessLogs
	dp, exists := meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"]
	if !exists { // Add defaultProviders.accessLogs if it does not exist
		meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"] = []string{"sentryflow"}
	} else { // Just add a new entry sentryflow if it exists
		dpSlice := dp.([]interface{}) // @todo find better solution for this
		duplicate := false
		for _, entry := range dpSlice {
			if entry == "sentryflow" {
				// If "sentryflow" already exists, do nothing
				log.Printf("[Patcher] istio-system/istio ConfigMap has " +
					"sentryflow under defaultProviders.accessLogs, ignoring... ")
				duplicate = true
				break
			}
		}

		// If "sentryflow" does not exist, append it
		if !duplicate {
			dpSlice = append(dpSlice, "sentryflow")
			meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"] = dpSlice
		}
	}

	// ExtensionProvider for our service
	eps := map[interface{}]interface{}{
		"name": "sentryflow",
		"envoyOtelAls": map[interface{}]interface{}{
			"service": "sentryflow.sentryflow.svc.cluster.local",
			"port":    config.GlobalCfg.OtelGRPCListenPort,
		},
	}

	// Work with extensionProviders
	ep, exists := meshConfig["extensionProviders"]
	if !exists {
		// Create extensionProviders as a slice containing only the eps map
		meshConfig["extensionProviders"] = []map[interface{}]interface{}{eps}
	} else {
		// Check if eps already exists in extensionProviders
		epSlice, ok := ep.([]interface{})
		if !ok {
			// handle the case where ep is not []interface{}
			log.Printf("[Patcher] istio-system/istio ConfigMap extensionProviders has unexpected type")
		}

		duplicate := false
		for _, entry := range epSlice {
			entryMap, ok := entry.(map[interface{}]interface{})
			if !ok {
				// handle the case where an entry is not map[interface{}]interface{}
				log.Printf("[Patcher] istio-system/istio ConfigMap extensionProviders entry has unexpected type")
			}
			if entryMap["name"] == eps["name"] {
				// If "sentryflow" already exists, do nothing
				log.Printf("[Patcher] istio-system/istio ConfigMap has sentryflow under extensionProviders, ignoring... ")
				duplicate = true
				break
			}
		}

		// Append eps to the existing slice
		if !duplicate {
			meshConfig["extensionProviders"] = append(ep.([]map[interface{}]interface{}), eps)
		}
	}

	// Update the ConfigMap data with the modified meshConfig
	updatedMeshConfig, err := yaml.Marshal(meshConfig)
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to marshal updated meshConfig to YAML: %v", err)
		return err
	}

	// Convert the []byte to string
	configMap.Data["mesh"] = string(updatedMeshConfig)

	// Preview changes, for debugging
	if config.GlobalCfg.Debug {
		log.Printf("[PATCH] Patching istio-system/istio ConfigMap as: \n%v", configMap)
	}

	// Patch the ConfigMap back to the cluster
	updatedConfigMap, err := kh.clientSet.CoreV1().
		ConfigMaps("istio-system").
		Update(context.Background(), configMap, v1.UpdateOptions{})
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to update configmap istio-system/istio :%v", err)
		return err
	}

	// Update successful
	if config.GlobalCfg.Debug {
		log.Printf("[Patcher] Updated istio-system/istio ConfigMap as: \n%v", updatedConfigMap)
	}
	return nil
}

// PatchNamespaces patches namespaces for adding istio injection
func (kh *K8sHandler) PatchNamespaces() error {
	// Get the list of namespaces
	namespaces, err := kh.clientSet.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to list namespaces: %v", err)
		return err
	}

	// Loop through each namespace and update it with the desired labels
	// @todo make this skip adding labeles to namespaces which are defined in the config
	for _, ns := range namespaces.Items {
		currentNs := ns

		// We are not going to inject sidecars to sentryflow namespace
		if currentNs.Name == "sentryflow" {
			continue
		}

		// Add istio-injection="enabled" for namespaces
		currentNs.Labels["istio-injection"] = "enabled"

		// Update the namespace in the cluster
		updatedNamespace, err := kh.clientSet.CoreV1().Namespaces().Update(context.TODO(), &currentNs, v1.UpdateOptions{
			FieldManager: "patcher",
		})
		if err != nil {
			log.Printf("[Patcher] Unable to update namespace %s: %v", currentNs.Name, err)
			return err
		}

		log.Printf("[Patcher] Updated Namespace: %s\n", updatedNamespace.Name)
	}

	return nil
}

// PatchRestartDeployments restarts the deployments in namespaces which were applied with "istio-injection": "enabled"
func (kh *K8sHandler) PatchRestartDeployments() error {
	// Get the list of all deployments in all namespaces
	deployments, err := kh.clientSet.AppsV1().Deployments("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to list deployments: %v", err)
		return err
	}

	// Iterate over each deployment and restart it
	for _, deployment := range deployments.Items {
		// We are not going to inject sidecars to sentryflow namespace
		if deployment.Namespace == "sentryflow" {
			continue
		}

		// Restart the deployment
		err := kh.restartDeployment(deployment.Namespace, deployment.Name)
		if err != nil {
			// Handle error
			log.Printf("[Patcher] Unable to restart deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
			continue
		}

		log.Printf("[Patcher] Deployment %s/%s restarted", deployment.Namespace, deployment.Name)
	}

	return nil
}

// restartDeployment performs a rolling restart for a deployment in the specified namespace
// @todo: fix this, this DOES NOT restart deployments
func (kh *K8sHandler) restartDeployment(namespace string, deploymentName string) error {
	deploymentClient := kh.clientSet.AppsV1().Deployments(namespace)

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
