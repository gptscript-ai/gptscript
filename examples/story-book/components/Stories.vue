<script setup lang="ts">
import { useMainStore } from '@/store'
import unmangleStoryName from '@/lib/unmangle'

const toast = useToast()
const store = useMainStore()

const pendingStories = computed(() => store.pendingStories)
const pendingStoryMessages = computed(() => store.pendingStoryMessages)
const stories = computed(() => store.stories)
onMounted(async () => store.fetchStories() )

const goToStory = (name: string) => {
    useRouter().push(`/story/${name}`)
}

const deleteStory = async (name: string) => {
    const response = await fetch(`/api/story/${name}`, { method: 'DELETE' })
    if (response.ok) {
        store.removeStory(name)
        toast.add({
            id: 'story-deleted',
            title: `${name} deleted`,
            description: 'The story has been deleted.',
            icon: 'i-heroicons-trash',
        })
    } else {
        toast.add({
            id: 'story-delete-failed',
            title: `Failed to delete ${name}`,
            description: 'The story could not be deleted.',
            icon: 'i-heroicons-x-mark',
        })
    }
}
</script>

<template>
    <div class="flex flex-col space-y-2">
        <New v-if="!Object.keys(pendingStories).length" class="w-full"/>
        <UPopover mode="hover" v-for="(_, name) in pendingStories" >
            <UButton truncate loading :key="name" size="lg" class="w-full text-xl" :label="name" icon="i-heroicons-book-open"/>

            <template #panel>
                <UCard class="p-4 w-[80vw] xl:w-[40vw]">
                    <h1 class="text-xl">Writing the perfect story...</h1>
                    <h2 class="text-zinc-400 mb-4">GPTScript is currently building the story you requested. You can see its progress below.</h2>
                    <pre class="h-[26vh] bg-zinc-950 px-6 text-white overflow-x-scroll rounded shadow">
                        <p v-for="message in pendingStoryMessages[name]">> {{ message }}</p>
                    </pre>
                </UCard>
            </template>
        </UPopover>
        
        <div class="w-full" v-for="story in stories" >
            <!-- The LLM likes to ocassionally, not always, generate folders with "-" and other times with " ", handle both cases for the button -->
            <UButton truncate :key="story" size="lg" class="w-5/6 text-xl" :label="unmangleStoryName(story)" icon="i-heroicons-book-open" @click="goToStory(story)"/>
            <UButton size="lg" variant="ghost" class="w-1/7 text-xl ml-4" icon="i-heroicons-trash" @click="deleteStory(story)"/>
        </div>

    </div>
</template>