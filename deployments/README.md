# KubeView - Helm Chart

Supplied is a Helm chart called `kubeview` to deploy and install KubeView into your cluster.

Install with the standatd Helm command:

```bash
cd deployments/helm
helm install ./kubeview --name kubeview -f myvalues.yaml
```