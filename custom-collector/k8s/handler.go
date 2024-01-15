package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Handler represents the k8s-api handler
type Handler struct {
	config       *rest.Config
	clientSet    *kubernetes.Clientset
	executionID  string
	curIndex     int
	listWatchers []*cache.ListWatch
}

// NewHandler creates a new Kubernetes handler object
func NewHandler() (*Handler, error) {
	h := &Handler{}
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

	// Initialize watchers for services and pods
	h.initWatchers(watchTargets)

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

		h.listWatchers = append(h.listWatchers, watcher)
	}
}

// initInformers initializes informers for services in
