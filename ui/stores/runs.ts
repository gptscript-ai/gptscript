import type { JsonDict } from '@/utils/object'
import { defineStore } from 'pinia'

export const useRuns = defineStore('runs', {
  state: () => {
    return {
      list: reactive<Run[]>([]),
      map: {} as Record<string, Run>
    }
  },

  getters: {
  },

  actions: {
    async find(id: string) {
      if ( this.map[id] ) {
        return this.map[id]
      }

      // const url = useRuntimeConfig().public.api
      // const { data } = (await useFetch(url+id) as any as {data: Ref<Gpt>})

      // this.list.push(data)
      // this.map[id] = data
      // return data
    },

    async findAll() {
      return this.list
    },

    async create(fileName: string, toolName: string, args: any, cache=true): Promise<Run> {
      let url = useRuntimeConfig().public.api + fileName.split('/').map(x => encodeURIComponent(x)).join('/') + '?async'

      if ( !cache ) {
        url = addParam(url, 'nocache', null)
      }

      if ( toolName ) {
        url = addParam(url, 'tool', toolName)
      }

      const res = await $fetch(url, {method: 'POST', body: args}) as {id: string}
      const id = res.id

      const obj: Run = reactive({
        id,
        state: 'creating',
        calls: []
      })

      this.list.push(obj)
      this.map[id] = obj
      console.log('Created', id)
      return obj
    }
  }
})
