<template>
  <div class="app">
    <b-navbar toggleable="md" type="dark" variant="dark">
      <b-navbar-toggle target="nav_collapse"></b-navbar-toggle>

      <b-navbar-brand class="logoText"><img src="./assets/logo.png" class="logo"> &nbsp;KubeView</b-navbar-brand>

      <b-collapse is-nav id="nav_collapse">
        <b-navbar-nav>
          <b-dropdown :text="namespace" variant="info">
            <b-dropdown-header>Pick namespace to show</b-dropdown-header>
            <b-dropdown-item @click="filter = ''; namespace = ns.metadata.name" v-for="ns in namespaces" :key="ns.metadata.uid" >{{ ns.metadata.name }}</b-dropdown-item>
          </b-dropdown>&nbsp;&nbsp;

            <datalist id="ns-list">
              <option v-for="ns in namespaces" :key="ns.metadata.uid" >{{ ns.metadata.name }}</option>
            </datalist>
        </b-navbar-nav>

        <b-navbar-nav>
          <b-form-input v-model="filter" @keyup.enter="$refs.viewer.refreshData(false)" class="filterBox" placeholder="filter..."></b-form-input>&nbsp;&nbsp;
          <b-button variant="info" @click="$refs.viewer.refreshData(false)">Refresh</b-button> &nbsp;&nbsp;
        </b-navbar-nav>

        <b-navbar-nav>
          <b-dropdown split :text="autoRefreshText" split-variant="light" variant="info">
            <b-dropdown-item @click="autoRefresh=0">Off</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=5">5 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=10">10 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=15">15 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=30">30 secs</b-dropdown-item>
            <b-dropdown-item @click="autoRefresh=60">60 secs</b-dropdown-item>
          </b-dropdown>
        </b-navbar-nav>
      </b-collapse>

      <b-navbar-nav class="ml-auto">
        <b-button variant="success" v-b-modal.aboutModal>Help</b-button>
      </b-navbar-nav>
    </b-navbar>

    <viewer :namespace="namespace" :filter="filter" :autoRefresh="autoRefresh" ref="viewer"></viewer>

    <b-modal id="aboutModal" title="Help with KubeView" header-bg-variant="info" header-text-variant="dark" ok-only>
      <div class="text-center">
        <img src="./assets/logo.png" width="100">

        <p>Kubeview displays the current state of your cluster. Each resource is represented by a different icon, click them to see information on the resource.</p>
        <p>Solid lines indicate that one resource <i>references</i> another in the direction of the arrow, a dashed line indicates that one resources <i>creates</i> another in the direction of the arrow.</p>

        <b-button href="https://github.com/REBELinBLLUE/kubeview" target="_blank" variant="success">GitHub Project</b-button>

        <p id="credit">Based on <a target="_blank" href="https://github.com/benc-uk/kubeview">KubeView by Ben Coleman</a></p>
      </div>
    </b-modal>
  </div>
</template>

<script>
import Viewer from './components/Viewer.vue'
import apiMixin from "./mixins/api.js";

export default {
  mixins: [ apiMixin ],

  components: {
    Viewer
  },

  computed: {
    autoRefreshText() {
      return this.autoRefresh ? `Auto Refresh: ${this.autoRefresh} secs` : "Auto Refresh: Off"
    }
  },

  data() {
    return {
      namespace: "",
      namespaces: [],
      filter: "",
      version: require('../package.json').version,
      autoRefresh: 0
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
          // FIXME: There has to be a cleaner way to do this, set the default namespace to the first namespace if default is not accessible
          let defaultNs = this.namespaces[0].metadata.name;
          this.namespaces.forEach((ns) => {
              if (ns.metadata.name == 'default') {
                  defaultNs = 'default';
              }
          })

          this.namespace = defaultNs
      })
  }
}
</script>

<style>
  body, html, .app {
    margin: 0;
    padding: 0;
    height: 100%
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
  #credit {
    margin: 15px auto 0px auto;
  }
</style>
