package k8s

import (
	"context"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"numbat/common"
)

// PatchIstioConfigMap patches the Istio's configmap for meshConfig
// This will make istio know that there is an exporter with envoyOtelAls
func (h *Handler) PatchIstioConfigMap() error {
	// Get the ConfigMap istio-system/istio
	configMap, err := h.clientSet.CoreV1().
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

	// Work with defaultProviders.accessLogs
	dp, exists := meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"]
	if !exists { // Add defaultProviders.accessLogs if it does not exist
		meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"] = []string{"boanlab-collector-1"}
	} else { // Just add a new entry boanlab-collector-1 if it exists
		dpSlice := dp.([]string)
		duplicate := false
		for _, entry := range dpSlice {
			if entry == "boanlab-collector-1" {
				// If "boanlab-collector-1" already exists, do nothing
				log.Printf("[Patcher] istio-system/istio ConfigMap has " +
					"boanlab-collector-1 under defaultProviders.accessLogs, ignoring... ")

				duplicate = true
				break
			}
		}

		// If "boanlab-collector-1" does not exist, append it
		if !duplicate {
			dpSlice = append(dpSlice, "boanlab-collector-1")
			meshConfig["defaultProviders"].(map[interface{}]interface{})["accessLogs"] = dpSlice
		}
	}

	// ExtensionProvider for our service
	eps := map[interface{}]interface{}{
		"name": "boanlab-collector-1",
		"envoyOtelAls": map[interface{}]interface{}{
			"service": "custom-collector.collector-1.svc.cluster.local",
			"port":    4317,
		},
	}

	// Work with extensionProviders
	ep, exists := meshConfig["extensionProviders"]
	if !exists {
		// Create extensionProviders as a slice containing only the eps map
		meshConfig["extensionProviders"] = []map[interface{}]interface{}{eps}
	} else {
		// Check if eps already exists in extensionProviders
		duplicate := false
		epSlice := ep.([]map[interface{}]interface{})
		for _, entry := range epSlice {
			if entry["name"] == eps["name"] {
				// If "boanlab-collector-1" already exists, do nothing
				log.Printf("[Patcher] istio-system/istio ConfigMap has " +
					"boanlab-collector-1 under extensionProviders, ignoring... ")

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
	log.Printf("[PATCH] Patching istio-system/istio ConfigMap as: \n%v", configMap)

	// Patch the ConfigMap back to the cluster
	updatedConfigMap, err := h.clientSet.CoreV1().
		ConfigMaps("istio-system").
		Update(context.Background(), configMap, v1.UpdateOptions{})
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to update configmap istio-system/istio :%v", err)
		return err
	}

	// Update successful
	log.Printf("[PATCH] Updated istio-system/istio ConfigMap as: \n%v", updatedConfigMap)
	return nil
}

// PatchNamespaces patches namespaces for adding istio injection
// The namespaces to ignore can be defined as IGNORE_NAMESPACES environment variable
func (h *Handler) PatchNamespaces() error {
	// Get the list of namespaces
	namespaces, err := h.clientSet.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to list namespaces: %v", err)
		return err
	}

	// Define the labels to add to each namespace
	labels := map[string]string{
		"istio-injection": "enabled",
	}

	// Loop through each namespace and update it with the desired labels
	for _, ns := range namespaces.Items {
		// Skip namespaces listed in the IGNORE_NAMESPACES environment variable
		if shouldIgnoreNamespace(ns.Name) {
			continue
		}

		// Update the labels for the namespace
		ns.Labels = labels

		// Update the namespace in the cluster
		updatedNamespace, err := h.clientSet.CoreV1().Namespaces().Update(context.TODO(), &ns, v1.UpdateOptions{
			FieldManager: "patcher",
		})
		if err != nil {
			log.Printf("[Patcher] Unable to update namespace %s: %v", ns.Name, err)
			return err
		}

		log.Printf("[Patcher] Updated Namespace: %s\n", updatedNamespace.Name)
	}

	return nil
}

// PatchRestartDeployments restarts the deployments in namespaces which were applied with "istio-injection": "enabled"
func (h *Handler) PatchRestartDeployments() error {
	// Get the list of all deployments in all namespaces
	deployments, err := h.clientSet.AppsV1().Deployments("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		// Handle error
		log.Fatalf("[Patcher] Unable to list deployments: %v", err)
		return err
	}

	// Iterate over each deployment and restart it
	for _, deployment := range deployments.Items {
		// Check if the namespace should be ignored
		if shouldIgnoreNamespace(deployment.Namespace) {
			continue
		}

		// Restart the deployment
		err := h.restartDeployment(deployment.Namespace, deployment.Name)
		if err != nil {
			// Handle error
			log.Printf("[Patcher] Unable to restart deployment %s in namespace %s: %v", deployment.Name, deployment.Namespace, err)
			continue
		}

		log.Printf("Deployment %s in namespace %s restarted", deployment.Name, deployment.Namespace)
	}

	return nil
}

// shouldIgnoreNamespace checks if a namespace should be ignored based on the IGNORE_NAMESPACES environment variable
func shouldIgnoreNamespace(namespace string) bool {
	for _, ignore := range common.Cfg.IgnoreNamespaces {
		if namespace == ignore {
			return true
		}
	}
	return false
}

// restartDeployment performs a rolling restart for a deployment in the specified namespace
func (h *Handler) restartDeployment(namespace string, deploymentName string) error {
	deploymentClient := h.clientSet.AppsV1().Deployments(namespace)

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
