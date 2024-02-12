import { defineStore } from 'pinia'

export const useContext = defineStore('context', {
  state: () => {
    return {
      mgmtSetup: false,
    }
  },

  getters: {
    baseUrl: () => {
      let base = ''

      if ( process.server ) {
        const headers = useRequestHeaders()

        base = `http://${ headers.host }`
      }

      base += '/v1'

      return base
    },
  },

  actions: {
    async setupMgmt() {
      if ( this.mgmtSetup ) {
        return
      }

      this.mgmtSetup = true
    },
  }
})
