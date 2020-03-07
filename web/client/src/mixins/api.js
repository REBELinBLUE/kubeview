const API_ENDPOINT = process.env.VUE_APP_API_ENDPOINT
const FILTERED_NS = [] //['prod', 'qa1', 'qa2', 'qa3', 'qa4', 'stage1', 'stage2', 'stage3', 'demo', 'integrity', 'uat', 'tools', 'devops']

export default {
  methods: {
    apiGetDataForNamespace(ns) {
      return fetch(`${API_ENDPOINT}/scrape/${ns}`)
        .then(resp => {
          if (resp.ok) {
            return resp.json();
          }

          throw resp
        })
        .catch(err => {
          // eslint-disable-next-line
          if (err.status == 403) {
            throw err
          }

          if (!err.text) {
            console.error(err)
            return
          }

          err.text().then(message => {
            console.log(`### API Error! ${message}`);
          })
        })
    },

    apiGetNamespaces() {
      return fetch(`${API_ENDPOINT}/namespaces`)
        .then(resp => {
          return resp.json();
        })
        .then(namespaces => {
            if (FILTERED_NS.length == 0) {
                return namespaces
            }

            return namespaces.filter(ns => FILTERED_NS.includes(ns.metadata.name))
        })
        .catch(err => {
          // eslint-disable-next-line
          console.log(`### API Error! ${err.toString()}`);
        })
    }
  }
}
