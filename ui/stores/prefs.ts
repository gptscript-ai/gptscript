import { defineStore } from 'pinia'
import { useStorage, type RemovableRef } from '@vueuse/core'

const prefix='gptscript-'

export const usePrefs = defineStore('prefs', {
  state: () => {
    return {
      debug: useStorage(`${prefix}debug`, false) as RemovableRef<boolean>,
      cache: useStorage(`${prefix}cache`, true) as RemovableRef<boolean>,
      expanded: <MapBool>{},
      nextCall: 1,
      callMap: {} as Record<string, string>,
      allExpanded: useStorage(`${prefix}expand-all`, false) as RemovableRef<boolean>,
    }
  },

  getters: {},
  actions: {
    expand(id: string) {
      this.expanded[id] = true
    },

    collapse(id: string) {
      this.expanded[id] = false
      this.allExpanded = false
    },

    expandAll(keys: string[]=[]) {
      this.allExpanded = true
      this.expanded = {}
      if ( keys.length ) {
        for ( const k of keys ) {
          this.expanded[k] = true
        }
      }
    },

    collapseAll() {
      this.allExpanded = false
      this.expanded = {}
    },

    toggleAll() {
      if ( this.allExpanded ) {
        this.collapseAll()
      } else {
        this.expandAll()
      }
    },

    mapCall(id: string) {
      if ( !this.callMap[id] ) {
        const k = 'call_' + this.nextCall++
        this.callMap[id] = k
      }

      return  this.callMap[id]
    }
  },
})
