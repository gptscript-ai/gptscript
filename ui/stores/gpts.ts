import { defineStore } from 'pinia'

export const useGpts = defineStore('gpts', {
  state: () => {
    return {
      list: [] as Gpt[],
      map: {} as Record<string, Gpt>
    }
  },

  getters: {
  },

  actions: {
    async find(id: string) {
      if ( this.map[id] ) {
        return this.map[id]
      }

      const url = useRuntimeConfig().public.api + id
      const data = reactive<Gpt>(await $fetch(url))

      this.list.push(data)
      this.map[id] = data
      return data
    },

    async listAll() {
      const url = useRuntimeConfig().public.api
      const data = await $fetch(url, {headers: {'X-Requested-With': 'fetch'} }) as string[]
      return reactive(data)
    }
  }
})
