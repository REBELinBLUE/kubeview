package main

//
// Kubeview API scraping service and client host
// Ben Coleman, July 2019, v1
//

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/benc-uk/go-starter/pkg/envhelper"

	"github.com/gorilla/mux"
	_ "github.com/joho/godotenv/autoload" // Autoloads .env file if it exists
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	healthy   = true                // Simple health flag
	version   = "0.1.13"            // App version number, set at build time with -ldflags "-X main.version=1.2.3"
	buildInfo = "No build details"  // Build details, set at build time with -ldflags "-X main.buildInfo='Foo bar'"
	clientset *kubernetes.Clientset // Clientset is global because I don't care
)

//
// Main entry point, will start HTTP service
//
func main() {
	klog.SetOutputBySeverity("INFO", os.Stdout)
	klog.SetOutputBySeverity("WARNING", os.Stdout)
	klog.SetOutputBySeverity("ERROR", os.Stderr)
	klog.SetOutputBySeverity("FATAL", os.Stderr)

	klog.Infof("### Kubeview v%v starting...", version)

	// Port to listen on, change the default as you see fit
	serverPort := envhelper.GetEnvInt("PORT", 8000)
	inCluster := envhelper.GetEnvBool("IN_CLUSTER", false)

	klog.Info("### Connecting to Kubernetes...")
	var config *rest.Config
	var err error

	// In cluster connect using in-cluster "magic", else build config from .kube/config file
	if inCluster {
		klog.Info("### Creating client in cluster mode")
		config, err = rest.InClusterConfig()
	} else {
		var kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config") // FIXME: Check for KUBECONFIG variable
		klog.Infof("### Creating client with config file: %s", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// We have to give up if we can't connect to Kubernetes
	if err != nil {
		panic(err.Error())
	}

	klog.Infof("### Connected to: %s", config.Host)

	// Create the clientset, which is our main interface to the Kubernetes API
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Use gorilla/mux for routing
	router := mux.NewRouter()

	// Add middleware for logging and CORS
	router.Use(starterMiddleware)

	// Application routes here
	router.HandleFunc("/healthz", routeHealthCheck)
	router.HandleFunc("/api/status", routeStatus)
	router.HandleFunc("/api/namespaces", routeGetNamespaces)
	router.HandleFunc("/api/scrape/{ns}", routeScrapeData)

	staticDirectory := envhelper.GetEnvString("STATIC_DIR", "./frontend")
	fileServer := http.FileServer(http.Dir(staticDirectory))

	router.PathPrefix("/js").Handler(http.StripPrefix("/", fileServer))
	router.PathPrefix("/css").Handler(http.StripPrefix("/", fileServer))
	router.PathPrefix("/img").Handler(http.StripPrefix("/", fileServer))
	router.PathPrefix("/favicon.png").Handler(http.StripPrefix("/", fileServer))

	// EVERYTHING else redirect to index.html
	router.NotFoundHandler = http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		http.ServeFile(resp, req, staticDirectory+"/index.html")
	})

	klog.Infof("### Serving static content from '%s'", staticDirectory)

	// Start server
	klog.Infof("### Server listening on %v", serverPort)

	err = http.ListenAndServe(fmt.Sprintf(":%d", serverPort), router)

	if err != nil {
		panic(err.Error())
	}
}

//
// Log all HTTP requests with client address, method and request URI
// Plus a cheap and dirty CORS enabler
//
func starterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		var lastPos int = strings.LastIndex(req.RemoteAddr, ":")

		resp.Header().Set("Access-Control-Allow-Origin", "*")
		klog.Infof("### %s %s %s", req.RemoteAddr[0:lastPos], req.Method, req.RequestURI)
		next.ServeHTTP(resp, req)
	})
}
