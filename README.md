# KubeView

> Fork of [KubeView by Ben Coleman](https://github.com/benc-uk/kubeview). I have forked the package as I am adding many new features specific to my needs.

Kubernetes cluster visualiser and visual explorer

KubeView displays what is happening inside a Kubernetes cluster, it maps out the API objects and how they are interconnected. Data is fetched real-time from the Kubernetes API. The status of some objects (Pods, ReplicaSets, Deployments) is colour coded red/green to represent their status and health

The app auto refreshes and dynamically updates the view as new data comes in or changes

Currently displays:

- Deployments
- ReplicaSets / StatefulSets / DaemonSets
- Pods
- ConfigMaps
- Secrets
- Endpoints
- Services
- Ingresses
- LoadBalancer IPs or Hostnames
- PersistentVolumeClaims
- PersistentVolumes
- StorageClasses

## Application Components

- **Client SPA** - Vue.js single page app. All visualisation, mapping & logic done here
- **API Server** - Scrapes Kubernetes API and presents it back out as a custom REST API. Also acts as HTTP serving host to the SPA. Written in Go

## Repo Details

This projects follows the 'Standard Go Project Layout' directory structure and naming conventions as described [here](https://github.com/golang-standards/project-layout)

- [/cmd/server](./cmd/server) - Source of the API server, written in Go
- [/web/client](./web/client) - Source of the client app, written in Vue.js
- [/deployments](./deployments) - Kubernetes deployment manifests and instructions
