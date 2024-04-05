// store/index.ts
import { defineStore } from 'pinia'

export const useMainStore = defineStore({
    id: 'main',
    state: () => ({
        pendingStories: {} as Record<string, EventSource>,
        pendingStoryMessages: {} as Record<string, string[]>,
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
        addPendingStoryMessage(name: string, message: string) {
            // implement a queue with a length of 10 that removes the oldest message when the length is reached
            if (!this.pendingStoryMessages[name]) {
                this.pendingStoryMessages[name] = []
            }
            this.pendingStoryMessages[name].push(message)
            if (this.pendingStoryMessages[name].length > 12) {
                this.pendingStoryMessages[name].shift()
            }
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
            if (Object.keys(this.pendingStories).length === 0) {
                this.addStories(await $fetch('/api/story') as string[])
            }
        }

    }
})