<template>
  <div class="app">
    <!-- FIXME: Move this to another component -->
    <b-navbar toggleable="md" type="dark" variant="dark">
      <b-navbar-toggle target="nav_collapse"></b-navbar-toggle>

      <b-navbar-brand class="logoText">
        <img src="./assets/logo.png" class="logo"> &nbsp;KubeView
      </b-navbar-brand>

      <b-collapse is-nav id="nav_collapse">
        <b-navbar-nav>
          <b-dropdown :text="namespace" variant="info">
            <b-dropdown-header>Pick namespace to show</b-dropdown-header>
            <b-dropdown-item @click="filter = ''; namespace = ns.metadata.name" v-for="ns in namespaces" :key="ns.metadata.uid" >{{ ns.metadata.name }}</b-dropdown-item>
          </b-dropdown>&nbsp;&nbsp;

          <datalist id="ns-list">
            <option v-for="ns in namespaces" :key="ns.metadata.uid" >{{ ns.metadata.name }}</option>
          </datalist>

          <b-form-input v-model="filter" @keyup.enter="$refs.viewer.refreshData(false)" class="filterBox" placeholder="filter..."></b-form-input>&nbsp;&nbsp;
        </b-navbar-nav>

        <b-navbar-nav class="refresh">
          <b-button variant="info" @click="$refs.viewer.refreshData(false)">Refresh</b-button> &nbsp;&nbsp;
          <b-dropdown split :text="autoRefreshText" split-variant="light" variant="info">
            <b-dropdown-item @click="autoRefresh=0">Off</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=2">2 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=5">5 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=10">10 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=15">15 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=30">30 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=60">60 secs</b-dropdown-item>
          </b-dropdown>
        </b-navbar-nav>

        <b-navbar-nav>
           <b-dropdown split :text="displayModeText" split-variant="light" variant="info">
             <b-dropdown-item @click="displayMode=0">Workloads</b-dropdown-item>
             <b-dropdown-item @click="displayMode=1">Nodes</b-dropdown-item>
           </b-dropdown>
         </b-navbar-nav>    
      </b-collapse>

      <b-navbar-nav class="ml-auto">
        <b-button variant="primary" v-b-modal.optionsModal>Settings</b-button>
        <b-button variant="success" v-b-modal.aboutModal>Help</b-button>
      </b-navbar-nav>
    </b-navbar>

    <viewer :options="options" :namespace="namespace" :filter="filter" :autoRefresh="autoRefresh" :displayMode="displayMode" ref="viewer" />

    <help />

    <!-- FIXME: Move this to another component -->
    <b-modal id="optionsModal" title="Settings" header-bg-variant="info" header-text-variant="dark" ok-only>
      <b-form>
        <b-form-group label="Configuration">
          <b-form-checkbox v-model="options.configmaps" name="configmaps"> Show ConfigMaps</b-form-checkbox>
          <b-form-checkbox v-model="options.secrets" name="secrets"> Show Secrets</b-form-checkbox>
        </b-form-group>

        <b-form-group label="Storage">
          <b-form-checkbox v-model="options.persistentvolumeclaims" name="persistentvolumeclaims"> Show PersistentVolumeClaims</b-form-checkbox>
          <b-form-checkbox v-model="options.persistentvolumes" name="persistentvolumes" v-b-tooltip.hover.left="'Requires Claims'"> Show PersistentVolumes</b-form-checkbox>
          <b-form-checkbox v-model="options.storageclasses" name="storageclasses" v-b-tooltip.hover.left="'Requires Volumes'"> Show StorageClasses</b-form-checkbox>
        </b-form-group>

        <b-form-group label="Networking">
          <b-form-checkbox v-model="options.services" name="services"> Show Services &amp; Endpoints</b-form-checkbox>
          <b-form-checkbox v-model="options.ingresses" name="ingresses" v-b-tooltip.hover.left="'Requires Services'"> Show Ingresses</b-form-checkbox>
          <b-form-checkbox v-model="options.ingress_tls" name="ingress_tls" v-b-tooltip.hover.left="'Requires Ingresses'"> Show Ingresses TLS Secrets</b-form-checkbox>
          <b-form-checkbox v-model="options.loadbalancers" name="loadbalancers" v-b-tooltip.hover.left="'Requires Services/Ingresses'"> Show LoadBalancers</b-form-checkbox>
        </b-form-group>

        <b-form-group label="Miscellaneous">
          <b-form-checkbox v-model="options.serviceaccounts" name="sa"> Show ServiceAccounts</b-form-checkbox>
        </b-form-group>
      </b-form>
    </b-modal>
  </div>
</template>

<script>
import Viewer from './components/Viewer.vue'
import Help from './components/Help.vue';
import apiMixin from "./mixins/api.js";

export default {
  mixins: [ apiMixin ],

  components: {
    Viewer,
    Help,
  },

  computed: {
    autoRefreshText() {
      return this.autoRefresh ? `Auto Refresh: ${this.autoRefresh} secs` : "Auto Refresh: Off"
    },
    displayModeText() {
      let mode = ['Workloads', 'Nodes']
      return `Top Level: ${mode[this.displayMode]}`
    }
  },

  data() {
    return {
      namespace: "",
      namespaces: [],
      filter: "",
      version: require('../package.json').version,
      autoRefresh: 0,
      displayMode: 0,
      options: {
        services: true,
        ingresses: true,
        ingress_tls: false,
        configmaps: false,
        secrets: false,
        serviceaccounts: false,
        persistentvolumeclaims: true,
        persistentvolumes: false,
        storageclasses: false,
        loadbalancers: true,
      }
    }
  },

  methods: {
    changeNS: function(evt) {
      this.filter = '';
      this.namespace = evt;
      this.$refs.ns.blur();
    }
  },

  mounted() {
    this
      .apiGetNamespaces()
      .then(data => {
        this.namespaces = data.sort((a, b) => {
          if (a.metadata.name > b.metadata.name) {
            return 1;
          }

          if (a.metadata.name < b.metadata.name) {
            return -1;
          }

          return 0
        })
      })
      .then(() => {
        // Set the default namespace to the first namespace if default is not accessible
        const defaultNamespaces = this.namespaces.filter(({ metadata }) => metadata.name === 'default')
        const defaultNs = defaultNamespaces.length ? defaultNamespaces[0].metadata.name : this.namespaces[0].metadata.name

        this.namespace = defaultNs
      })

    this.displayMode = 0
    this.autoRefresh = 0
  }
}
</script>

<style>
  body, html, .app {
    margin: 0;
    padding: 0;
    height: 100%
  }
  .ml-auto button {
    margin-right: 10px;
  }
  .refresh {
    margin-left: 10px;
  }
  .logo {
    height: 45px;
  }
  .logoText {
    font-size: 30px !important;
  }
  .filterBox {
    font-size: 120%;
    width: 100px;
  }
</style>
