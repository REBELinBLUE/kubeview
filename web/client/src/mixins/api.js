const API_ENDPOINT = process.env.VUE_APP_API_ENDPOINT

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
          if (!window.FILTER_NAMESPACES || window.FILTER_NAMESPACES.length == 0) {
              return namespaces
          }

          return namespaces.filter(ns => window.FILTER_NAMESPACES.includes(ns.metadata.name))
        })
        .catch(err => {
          // eslint-disable-next-line
          console.log(`### API Error! ${err.toString()}`);
        })
    }
  }
}
