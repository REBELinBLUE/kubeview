package main

//
// Basic REST API microservice, template/reference code
// Ben Coleman, July 2019, v1
//

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/gorilla/mux"
	"k8s.io/klog"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//
// Simple health check endpoint, returns 204 when healthy
//
func routeHealthCheck(resp http.ResponseWriter, req *http.Request) {
	if healthy {
		resp.WriteHeader(http.StatusNoContent)
		return
	}

	resp.WriteHeader(http.StatusServiceUnavailable)
}

//
// Return status information data - Remove if you like
//
func routeStatus(resp http.ResponseWriter, req *http.Request) {
	type status struct {
		Healthy    bool   `json:"healthy"`
		Version    string `json:"version"`
		BuildInfo  string `json:"buildInfo"`
		Hostname   string `json:"hostname"`
		OS         string `json:"os"`
		Arch       string `json:"architecture"`
		CPU        int    `json:"cpuCount"`
		GoVersion  string `json:"goVersion"`
		ClientAddr string `json:"clientAddress"`
		ServerHost string `json:"serverHost"`
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "hostname not available"
	}

	currentStatus := status{
		Healthy:    healthy,
		Version:    version,
		BuildInfo:  buildInfo,
		Hostname:   hostname,
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		CPU:        runtime.NumCPU(),
		ClientAddr: req.RemoteAddr,
		ServerHost: req.Host,
	}

	statusJSON, err := json.Marshal(currentStatus)
	if err != nil {
		http.Error(resp, "Failed to get status", http.StatusInternalServerError)
		return
	}

	resp.Header().Add("Content-Type", "application/json")
	resp.Write(statusJSON)
}

// Data struct to hold our returned data
type scrapeData struct {
	Pods                   []apiv1.Pod                   `json:"pods"`
	Services               []apiv1.Service               `json:"services"`
	Endpoints              []apiv1.Endpoints             `json:"endpoints"`
	PersistentVolumes      []apiv1.PersistentVolume      `json:"persistentvolumes"`
	PersistentVolumeClaims []apiv1.PersistentVolumeClaim `json:"persistentvolumeclaims"`
	Deployments            []appsv1.Deployment           `json:"deployments"`
	DaemonSets             []appsv1.DaemonSet            `json:"daemonsets"`
	ReplicaSets            []appsv1.ReplicaSet           `json:"replicasets"`
	StatefulSets           []appsv1.StatefulSet          `json:"statefulsets"`
	Ingresses              []v1beta1.Ingress             `json:"ingresses"`
	ConfigMaps             []apiv1.ConfigMap             `json:"configmaps"`
	Secrets                []apiv1.Secret                `json:"secrets"`
	StorageClasses         []storagev1.StorageClass      `json:"storageclasses"`
	ServiceAccounts        []apiv1.ServiceAccount        `json:"serviceaccounts"`
	Nodes                  []apiv1.Node                  `json:"nodes"`
}

// GetNamespaces - Return list of all namespaces in cluster
func routeGetNamespaces(w http.ResponseWriter, r *http.Request) {
	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})

	if err != nil {
		klog.Errorf("### Kubernetes API error - %s", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	namespacesJSON, _ := json.Marshal(namespaces.Items)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")

	w.Write(namespacesJSON)
}

// ScrapeData - Return aggregated data from loads of different Kubernetes object types
func routeScrapeData(w http.ResponseWriter, r *http.Request) {
	klog.SetOutput(os.Stdout)
	params := mux.Vars(r)
	namespace := params["ns"]

	// If deployments, daemonsets, replicasets, statefulsets or pods can't be listed there is not much point continuing

	deployments, err := clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{})

	if err != nil {
		klog.Errorf("### Kubernetes API error - %s", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(metav1.ListOptions{})

	if err != nil {
		klog.Errorf("### Kubernetes API error - %s", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	replicasets, err := clientset.AppsV1().ReplicaSets(namespace).List(metav1.ListOptions{})

	if err != nil {
		klog.Errorf("### Kubernetes API error - %s", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(metav1.ListOptions{})

	if err != nil {
		klog.Errorf("### Kubernetes API error - %s", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})

	if err != nil {
		klog.Errorf("### Kubernetes API error - %s", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// If the remaining resource types can't be listed it doesn't matter, handle it on the frontend
	services, err := clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	endpoints, err := clientset.CoreV1().Endpoints(namespace).List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	pvs, err := clientset.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	ingresses, err := clientset.ExtensionsV1beta1().Ingresses(namespace).List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	storageclasses, err := clientset.StorageV1().StorageClasses().List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	serviceaccounts, err := clientset.CoreV1().ServiceAccounts(namespace).List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("### Kubernetes API error - %s", err.Error())
	}

	// Remove and hide Helm v3 release secrets, we're never going to show them
	secrets.Items = filterSecrets(secrets.Items, func(v apiv1.Secret) bool {
		return !strings.HasPrefix(v.ObjectMeta.Name, "sh.helm.release")
	})

	// Obfuscate & remove secret values
	for _, secret := range secrets.Items {
		// Inside 'last-applied-configuration'
		if secret.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] != "" {
			secret.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = "__VALUE REDACTED__"
		}

		for key := range secret.Data {
			secret.Data[key] = []byte("__VALUE REDACTED__")
		}
	}

	scrapeResult := scrapeData{
		Pods:                   pods.Items,
		Services:               services.Items,
		Endpoints:              endpoints.Items,
		PersistentVolumes:      pvs.Items,
		PersistentVolumeClaims: pvcs.Items,
		Deployments:            deployments.Items,
		DaemonSets:             daemonsets.Items,
		ReplicaSets:            replicasets.Items,
		StatefulSets:           statefulsets.Items,
		Ingresses:              ingresses.Items,
		ConfigMaps:             configmaps.Items,
		Secrets:                secrets.Items,
		StorageClasses:         storageclasses.Items,
		ServiceAccounts:        serviceaccounts.Items,
		Nodes:                  nodes.Items,
	}

	scrapeResultJSON, _ := json.Marshal(scrapeResult)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")

	w.Write([]byte(scrapeResultJSON))
}

//
// Filter a slice of Secrets
//
func filterSecrets(secretList []apiv1.Secret, f func(apiv1.Secret) bool) []apiv1.Secret {
	newSlice := make([]apiv1.Secret, 0)

	for _, secret := range secretList {
		if f(secret) {
			newSlice = append(newSlice, secret)
		}
	}

	return newSlice
}
