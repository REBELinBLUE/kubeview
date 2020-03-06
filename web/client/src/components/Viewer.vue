<template>
  <div id="viewwrap">
    <div id="mainview" ref="mainview"></div>

    <loading v-if="loading"></loading>

    <transition name="slide-fade">
      <infobox v-if="infoBoxData" :nodeData="infoBoxData" @hideInfoBox="infoBoxData = null" @fullInfo="showFullInfo"></infobox>
    </transition>

    <b-modal centered :title="fullInfoTitle" ref="fullInfoModal" ok-only scrollable size="lg" body-class="fullInfoBody">
      <pre>{{ fullInfoYaml }}</pre>
    </b-modal>
  </div>
</template>

<script>
import apiMixin from "../mixins/api.js"
import utils from "../mixins/utils.js"
import InfoBox from "./InfoBox"
import Loading from "./Loading"

import yaml from 'js-yaml'
import VueTimers from 'vue-timers/mixin'
import cytoscape from 'cytoscape'

// Urgh, gotta have this here, putting into data, causes weirdness
let cy

export default {
  mixins: [ apiMixin, utils, VueTimers ],

  components: {
    'infobox': InfoBox,
    'loading': Loading
  },

  props: [ 'namespace', 'filter', 'autoRefresh', 'rootType' ],

  data() {
    return {
      apiData: null,
      infoBoxData: null,
      fullInfoYaml: null,
      fullInfoTitle: "",
      loading: false,
    }
  },

  // VueTimers mixin is pretty sweet
  timers: {
    timerRefresh: { time: 60000, autostart: false, repeat: true }
  },

  watch: {
    namespace() {
      this.refreshData(false)
    },

    autoRefresh() {
      this.$timer.stop('timerRefresh')

      if (this.autoRefresh > 0) {
        this.timers.timerRefresh.time = this.autoRefresh * 1000
        this.$timer.start('timerRefresh')
      }
    }
  },

  methods: {
    //
    // Display the detail info dialog with YAML version of the selected object
    //
    showFullInfo() {
      let sourceCopy = {}
      Object.assign(sourceCopy, this.infoBoxData.sourceObj);

      // FIXME: This is a bad way to do it and should be done on the API level
      if (sourceCopy.type) {
        if (sourceCopy.type == "kubernetes.io/tls") {
          sourceCopy.data['tls.key'] = '<REDACTED>'
        } else if (sourceCopy.type == "Opaque") { // FIXME: Think of a better way to do this
          if (sourceCopy.data.role_id) {
            sourceCopy.data.role_id = '<REDACTED>'
          }

          if (sourceCopy.data.secret_id) {
            sourceCopy.data.secret_id = '<REDACTED>'
          }
        }
      }

      if (sourceCopy.metadata.annotations) {
        delete sourceCopy.metadata.annotations['kubectl.kubernetes.io/last-applied-configuration']
        if (Object.keys(sourceCopy.metadata.annotations).length == 0) {
          delete sourceCopy.metadata.annotations
        }
      }

      this.fullInfoYaml = yaml.safeDump(sourceCopy)
      this.fullInfoTitle = `${this.infoBoxData.type}: ${sourceCopy.metadata.name}`

      this.$refs.fullInfoModal.show()
    },

    //
    // Called by the auto refresh timer, invokes a 'soft' refresh
    //
    timerRefresh() {
      this.refreshData(true)
    },

    //
    // Called to reload data from the API and display it
    //
    refreshData(soft = false) {
      if (!this.namespace) {
        return
      }

      // Soft refresh will not redraw/refresh nodes if no changes
      if (!soft) {
        cy.remove("*")
        this.loading = true
      }

      this
        .apiGetDataForNamespace(this.namespace)
        .then(newData => {
          if (!newData) {
            return
          }

          let changed = true
          if (soft) {
            changed = this.detectChange(newData)
          }

          this.apiData = newData

          if (changed) {
            this.typeIndexes = []
            cy.remove("*")
            this.infoBoxData = false
            this.refreshNodes()
          }

          this.loading = false
        })
        .catch(err => {
          err.text().then(message => {
            this.loading = false
            alert(message)
          }).catch(err => {
            console.error(err);
          })
        })
    },

    //
    // On a soft refresh, this detects changes between old & new data
    //
    detectChange(data) {
      if (!this.apiData) {
        return false
      }

      // Scan new data, match with old objects and check resourceVersion changes
      for (let type in data) {
        for (let obj of data[type]) {
          // We have to skip these objects, the resourceVersion is constantly shifting
          if (obj.metadata.selfLink == '/api/v1/namespaces/kube-system/endpoints/kube-controller-manager' ||
              obj.metadata.selfLink == '/api/v1/namespaces/kube-system/endpoints/kube-scheduler'
          ) {
            continue
          }

          let oldObj = this.apiData[type].find(o => o.metadata.uid == obj.metadata.uid)
          if (!oldObj || (oldObj.metadata.resourceVersion != obj.metadata.resourceVersion)) {
            return true
          }
        }
      }

      // Scan old data and look for missing objects, which means they are deleted
      for (let type in this.apiData) {
        for (let obj of this.apiData[type]) {
          let newObj = data[type].find(o => o.metadata.uid == obj.metadata.uid)
          if (!newObj) {
            return true
          }
        }
      }

      return false
    },

    //
    // Some objects are colour coded by status
    //
    calcStatus(kubeObj) {
      let status = 'grey'

      try {
        if (kubeObj.metadata.selfLink.startsWith(`/apis/apps/v1/namespaces/${this.namespace}/deployments/`)) {
          status = 'red'

          let cond = kubeObj.status.conditions.find(c => c.type == 'Available') || {}
          if (cond.status == "True") {
            status = 'green'
          }
        }

        if (kubeObj.metadata.selfLink.startsWith(`/apis/apps/v1/namespaces/${this.namespace}/replicasets/`) ||
          kubeObj.metadata.selfLink.startsWith(`/apis/apps/v1/namespaces/${this.namespace}/statefulsets/`)
        ) {
          status = 'green'

          if (kubeObj.status.replicas != kubeObj.status.readyReplicas) {
            status = 'red'
          }
        }

        if (kubeObj.metadata.selfLink.startsWith(`/apis/apps/v1/namespaces/${this.namespace}/daemonsets/`)) {
          status = 'green'

          if (kubeObj.status.numberReady != kubeObj.status.desiredNumberScheduled) {
            status = 'red'
          }
        }

        if (kubeObj.metadata.selfLink.startsWith(`/api/v1/namespaces/${this.namespace}/pods/`)) {
          let cond = {}

          if (kubeObj.status && kubeObj.status.conditions) {
            cond = kubeObj.status.conditions.find(c => c.type == 'Ready')
          }

          if (cond && cond.status == "True") {
            status = 'green'
          }

          if (kubeObj.status.phase == 'Failed' || kubeObj.status.phase == 'CrashLoopBackOff') {
            status = 'red'
          } else if (kubeObj.status.phase == 'Succeeded') {
            status = 'green'
          }
        }
      } catch(err) {
        console.log(`### Problem with calcStatus for ${kubeObj.metadata.selfLink}`);
      }

      return status
    },

    //
    // Convience method to add ReplicaSets / DaemonSets / StatefulSets
    //
    addSet(type, kubeObjs) {
      for (let obj of kubeObjs) {
        if (!this.filterShowNode(obj)) {
          continue
        }

        let objId = `${type}_${obj.metadata.name}`

        // This skips and hides sets without any replicas
        if (obj.status) {
          if (obj.status.replicas == 0 || obj.status.desiredNumberScheduled == 0) {
            continue;
          }
        }

        // Add special "group" node for the set
        this.addGroup(type, obj.metadata.name)

        // Add set node and link it to the group
        this.addNode(obj, type, this.calcStatus(obj))

        // Find all owning deployments of this set (if any)
        for (let ownerRef of obj.metadata.ownerReferences || []) {
          // Skip owners that aren't deployments (like operators and custom objects)
          if (ownerRef.kind.toLowerCase() !== 'deployment') {
            continue;
          }

          // Link set up to the deployment
          this.addLink(`${ownerRef.kind}_${ownerRef.name}`, objId, 'creates')
        }
      }
    },

    //
    // The core processing logic is here, add objects to layout
    //
    refreshNodes() {
      // Add deployments
      for (let deploy of this.apiData.deployments || []) {
        if (!this.filterShowNode(deploy)) {
          continue
        }

        this.addNode(deploy, 'Deployment', this.calcStatus(deploy))
      }

      // The 'sets' - ReplicaSets / DaemonSets / StatefulSets
      this.addSet('ReplicaSet', this.apiData.replicasets || [])
      this.addSet('StatefulSet', this.apiData.statefulsets || [])
      this.addSet('DaemonSet', this.apiData.daemonsets || [])

      // Add pods
      for (let pod of this.apiData.pods || []) {
        if (!this.filterShowNode(pod)) {
          continue
        }

        // Add pods to containing group (ReplicaSet, DaemonSet, StatefulSet) that 'owns' them
        if (pod.metadata.ownerReferences) {
          // Most pods have owning set (rs, ds, sts) so are in a group
          let owner = pod.metadata.ownerReferences[0];
          let groupId = `grp_${owner.kind}_${owner.name}`

          this.addNode(pod, 'Pod', this.calcStatus(pod), groupId)
        } else {
          // Naked pods don't go into groups
          this.addNode(pod, 'Pod', this.calcStatus(pod))
        }

        // FIXME: Refactor, lots of duplication
        const containers = (pod.spec.initContainers || []).concat(pod.spec.containers || []);
        for (let container of containers) {
          for (let envFrom of container.envFrom || []) {
            if (envFrom.configMapRef) {
              let configmap = (this.apiData.configmaps || []).find(p => p.metadata.name == envFrom.configMapRef.name);
              if (!configmap) {
                continue;
              }

              this.addNode(configmap, 'ConfigMap')
              this.addLink(`Pod_${pod.metadata.name}`, `ConfigMap_${envFrom.configMapRef.name}`, 'references')
            }

            if (envFrom.secretRef) {
              let secret = (this.apiData.secrets || []).find(p => p.metadata.name == envFrom.secretRef.name);
              if (!secret) {
                continue;
              }

              this.addNode(secret, 'Secret')
              this.addLink(`Pod_${pod.metadata.name}`, `Secret_${envFrom.secretRef.name}`, 'references')
            }
          }

          for (let env of container.env || []) {
            if (!env.valueFrom) {
              continue;
            }

            if (env.valueFrom.configMapKeyRef) {
              let configmap = (this.apiData.configmaps || []).find(p => p.metadata.name == env.valueFrom.configMapKeyRef.name);
              if (!configmap) {
                continue;
              }

              this.addNode(configmap, 'ConfigMap')
              this.addLink(`Pod_${pod.metadata.name}`, `ConfigMap_${env.valueFrom.configMapKeyRef.name}`, 'references')
            }

            if (env.valueFrom.secretKeyRef) {
              let secret = (this.apiData.secrets || []).find(p => p.metadata.name == env.valueFrom.secretKeyRef.name);
              if (!secret) {
                continue;
              }

              this.addNode(secret, 'Secret')
              this.addLink(`Pod_${pod.metadata.name}`, `Secret_${env.valueFrom.secretKeyRef.name}`, 'references')
            }
          }
        }

        // Add PVCs linked to Pod
        for (let vol of pod.spec.volumes || []) {
          if (vol.persistentVolumeClaim) {
            let pvc = (this.apiData.persistentvolumeclaims || []).find(p => p.metadata.name == vol.persistentVolumeClaim.claimName)
            if (!pvc) {
              continue;
            }

            this.addNode(pvc, 'PersistentVolumeClaim')
            this.addLink(`Pod_${pod.metadata.name}`, `PersistentVolumeClaim_${vol.persistentVolumeClaim.claimName}`, 'references')

            let pv = (this.apiData.persistentvolumes || []).find(p => p.spec.claimRef.uid == pvc.metadata.uid);
            if (!pv) {
              continue;
            }

            this.addNode(pv, 'PersistentVolume')
            this.addLink(`PersistentVolume_${pv.metadata.name}`, `PersistentVolumeClaim_${vol.persistentVolumeClaim.claimName}`, 'creates')

            let sc = (this.apiData.storageclasses || []).find(p => p.metadata.name == pv.spec.storageClassName)
            if (!sc) {
                continue;
            }

            this.addNode(sc, 'StorageClass')
            this.addLink(`PersistentVolume_${pv.metadata.name}`, `StorageClass_${sc.metadata.name}`, 'references')

          }

          if (vol.configMap) {
            let configmap = (this.apiData.configmaps || []).find(p => p.metadata.name == vol.configMap.name);
            if (!configmap) {
              continue;
            }

            this.addNode(configmap, 'ConfigMap')
            this.addLink(`Pod_${pod.metadata.name}`, `ConfigMap_${vol.configMap.name}`, 'references')
          }

          if (vol.secret) {
            let secret = (this.apiData.secrets || []).find(p => p.metadata.name == vol.secret.secretName);
            if (!secret || secret.type == "kubernetes.io/service-account-token") { // FIXME: What about showing this if it isn't default?
              continue;
            }

            this.addNode(secret, 'Secret')
            this.addLink(`Pod_${pod.metadata.name}`, `Secret_${vol.secret.secretName}`, 'references')
          }
        }

        // FIXME: What about env linked secrets and configMaps

        // Find all owning sets of this pod
        for (let ownerRef of pod.metadata.ownerReferences || []) {
          // Link pod up to the owning set/group
          this.addLink(`${ownerRef.kind}_${ownerRef.name}`, `Pod_${pod.metadata.name}`, 'creates')
        }
      }

      // Find all services and endpoints
      for (let svc of this.apiData.services || []) {
        if (!this.filterShowNode(svc)) {
          continue
        }

        let serviceId = `Service_${svc.metadata.name}`

        if (svc.metadata.name == 'kubernetes') {
          continue
        }

        // Find matching endpoint, and merge subsets into service
        let ep = this.apiData.endpoints.find(ep => ep.metadata.name == svc.metadata.name)

        this.addNode(svc, 'Service')
        this.addNode(ep, 'Endpoints');
        this.addLink(serviceId, `Endpoints_${ep.metadata.name}`, 'creates')

        for (let subset of ep.subsets || []) {
          let addresses = (subset.addresses || [])

          for (let address of addresses || []) {
            if (!address.targetRef || address.targetRef.kind != "Pod") {
              continue
            }

            this.addLink(`Endpoints_${ep.metadata.name}`, `Pod_${address.targetRef.name}`, 'references')
          }
        }

        // Find all external addresses of service, and add them
        // For this we create a pseudo-object
        for (let lb of svc.status.loadBalancer.ingress || []) {
          // Fake Kubernetes object to display the LoadBalancers
          let ipObj = { metadata: { name: lb.ip || lb.hostname } }

          this.addNode(ipObj, 'LoadBalancer')
          this.addLink(`Service_${svc.metadata.name}`, `LoadBalancer_${ipObj.metadata.name}`, 'references')
        }
      }

      // Add Ingresses and link to Services
      for (let ingress of this.apiData.ingresses || []) {
        if (!this.filterShowNode(ingress)) {
          continue
        }

        this.addNode(ingress, 'Ingress')

        // Find all external addresses of ingresses, and add them
        for (let lb of ingress.status.loadBalancer.ingress || []) {
          // Fake Kubernetes object to display the LoadBalancers
          let ipObj = { metadata: { name: lb.ip || lb.hostname } }

          this.addNode(ipObj, 'LoadBalancer')
          this.addLink(`Ingress_${ingress.metadata.name}`, `LoadBalancer_${ipObj.metadata.name}`, 'references')
        }

        // Ingresses joined to Services by the rules
        for (let rule of ingress.spec.rules || []) {
          if (!rule.http.paths) {
            continue
          }

          for (let path of rule.http.paths || []) {
            let serviceName = path.backend.serviceName

            this.addLink(`Ingress_${ingress.metadata.name}`, `Service_${serviceName}`, 'references')
          }
        }

        // Ingress tls secrets
        for (let tls of ingress.spec.tls || []) {
          let secret = this.apiData.secrets.find(p => p.metadata.name == tls.secretName);
          if (!secret) {
            continue;
          }

          this.addNode(secret, 'Secret')
          this.addLink(`Ingress_${ingress.metadata.name}`, `Secret_${tls.secretName}`, 'references')
        }
      }

      // Finally done! Call re-layout
      this.relayout()
    },

    //
    // Relayout nodes and display them
    //
    relayout() {
      cy.resize()

      // Use breadthfirst with Deployments or DaemonSets or StatefulSets at the root
      cy.layout({
        name: 'breadthfirst',
        roots: cy.nodes(`[type="Deployment"],[type="DaemonSet"],[type="StatefulSet"]`),
        nodeDimensionsIncludeLabels: true,
        spacingFactor: 1
      }).run()
    },

    //
    // Add node to the Cytoscape graph
    //
    addNode(node, type, status = '', groupId = null) {
      try {
        const icons = {
          Deployment:               'deploy',
          ReplicaSet:               'rs',
          StatefulSet:              'sts',
          DaemonSet:                'ds',
          Pod:                      'pod',
          Service:                  'svc',
          LoadBalancer:             'ip',
          Ingress:                  'ing',
          PersistentVolumeClaim:    'pvc',
          ConfigMap:                'cm',
          Secret:                   'secret',
          PersistentVolume:         'pv',
          StorageClass:             'sc',
          Endpoints:                'ep',
        }

        const icon = icons[type] ? icons[type] : 'default';

        // Trim long names for labels, and get pod's hashed generated name suffix
        let label = node.metadata.name.substr(0, 24)
        if (label != node.metadata.name) {
            label += '…'
        }

        if (type == "Pod") {
          let podName = node.metadata.name.replace(node.metadata.generateName, '')
          label = podName || node.status.podIP || ""
        }

        //console.log(`### Adding: ${type} -> ${node.metadata.name || node.metadata.selfLink}`);
        cy.add({
          data: {
            id: `${type}_${node.metadata.name}`,
            label,
            icon,
            sourceObj: node,
            type,
            parent: groupId,
            status,
            name: node.metadata.name
          }
        })
      } catch(e) {
        if (e.message && e.message.includes('Can not create second element')) {
            return
        }

        console.error(`### Unable to add node: ${node.metadata.name || node.metadata.selfLink}`);
      }
    },

    //
    // Link two nodes together
    //
    addLink(sourceId, targetId, direction) {
      try {
        // This is the syntax Cytoscape uses for creating links
        cy.add({
          data: {
            id: `${sourceId}___${targetId}`,
            source: sourceId,
            target: targetId,
            direction
          }
        })
      } catch(e) {
        if (e.message && e.message.includes('Can not create second element')) {
            return
        }

        console.error(`### Unable to add link: ${sourceId} to ${targetId}`);
      }
    },

    //
    // A group is like a container, currently only used to hold Pods
    //
    addGroup(type, name) {
      try {
        cy.add({
          classes:['grp'],
          data: {
            id: `grp_${type}_${name}`,
            label: name,
            name
          }
        })
      } catch(e) {
        if (e.message && e.message.includes('Can not create second element')) {
            return
        }

        console.error(`### Unable to add group: ${name}`);
      }
    },

    //
    // Filter out nodes, called before adding/processing them
    //
    filterShowNode(node) {
      if (!this.filter || this.filter.length <= 0) {
        return true
      }

      if (node.metadata.name.includes(this.filter)) {
        return true
      }

      for (let labelName in node.metadata.labels) {
        if (labelName.includes(this.filter)) {
          return true
        }

        if (node.metadata.labels[labelName].includes(this.filter)) {
          return true
        }
      }

      return false
    }
  },

  //
  // Init component and set things up
  //
  mounted: function() {
    // Create cytoscape, this bad boy is why we're here
    cy = cytoscape({
      container: this.$refs.mainview,
      wheelSensitivity: 0.1,
      maxZoom: 5,
      minZoom: 0.2,
      selectionType: 'single'
    })

    // Styling cytoscape to look good, stylesheets are held as JSON external
    cy.style().selector('node[icon]').style(require('../assets/styles/node.json'));
    cy.style().selector('node[icon]').style("background-image", ele => ele.data('status') ? `img/res/${ele.data('icon')}-${ele.data('status')}.svg` : `img/res/${ele.data('icon')}.svg`)
    cy.style().selector('.grp').style(require('../assets/styles/grp.json'));
    cy.style().selector('edge[direction="references"]').style(require('../assets/styles/references.json'));
    cy.style().selector('edge[direction="creates"]').style(require('../assets/styles/creates.json'));
    cy.style().selector('node:selected').style({
      'border-width': '4',
      'border-color': 'rgb(0, 120, 215)'
    });

    // Click/select event opens the infobox
    cy.on('select', evt => {
      // Only work with nodes
      if (evt.target.isNode()) {
        // Force selection of single nodes only
        if (cy.$('node:selected').length > 1) {
          cy.$('node:selected')[0].unselect();
        }

        if (evt.target.hasClass('grp')) {
          return false
        }

        this.infoBoxData = evt.target.data()
      }
    })

    // Only sensible way I could find to hide the info box when unselecting
    cy.on('click tap', evt => {
      if (!evt.target.length && this.infoBoxData) {
        this.infoBoxData = false;
      }
    })

    // Initial load of everything ...
    this.refreshData()
  }
}
</script>

<style>
  #viewwrap {
    height: calc(100% - 67px)
  }

  /* Style the cytoscape canvas */
  #mainview {
    width: 100%;
    background-color: #333;
    height: 100%;
    box-shadow: inset 0 0 20px #000000;
  }

  .fullInfoBody {
    color: #28c8e4;
    background-color: #111;
  }

  .slide-fade-enter-active {
    transition: all .3s ease;
  }
  .slide-fade-leave-active {
    transition: all .3s cubic-bezier(1.0, 0.5, 0.8, 1.0);
  }
  .slide-fade-enter, .slide-fade-leave-to {
    transform: translateY(20px);
    opacity: 0;
  }
</style>
