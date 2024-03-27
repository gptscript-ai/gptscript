// store/index.ts
import { defineStore } from 'pinia'

export const useMainStore = defineStore({
    id: 'main',
    state: () => ({
        pendingStories: {} as Record<string, EventSource>,
        stories: [] as string[]
    }),
    actions: {
        addPendingStory(name: string, es: EventSource) {
            this.pendingStories[name] = es
        },
        removePendingStory(name: string) {
            this.pendingStories[name].close()
            delete this.pendingStories[name]
        },
        addStory(name: string) {
            this.stories.push(name)
        },
        addStories(names: string[]) {
            // only push if the story is not already in the list
            names.forEach(name => {
                if (!this.stories.includes(name)) {
                    this.stories.push(name)
                }
            })
        },
        removeStory(name: string) {
            this.stories = this.stories.filter(s => s !== name)
        },
        async fetchStories() {
            this.addStories(await $fetch('/api/story') as string[])
        }
    }
})