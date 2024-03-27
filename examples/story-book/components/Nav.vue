<script setup lang="ts">
import { useMainStore } from '@/store'

const isMenuOpen = ref(false)
const toast = useToast()
const store = useMainStore()

const pendingStories = computed(() => store.pendingStories)
const stories = computed(() => store.stories)
onMounted(async () => store.fetchStories() )

const goToStory = (name: string) => {
    useRouter().push(`/story/${name}`)
    isMenuOpen.value = false
}

const goToHome = () => {
    
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
  <div>
    <div class="flex space-x-4">
        <DisplayMode />
        <UButton icon="i-heroicons-bars-3" @click="isMenuOpen = true" />
    </div>

    <USlideover v-model="isMenuOpen">
        <UCard class="flex flex-col flex-1" :ui="{ body: { base: 'flex-1' }, ring: '', divide: 'divide-y divide-gray-100 dark:divide-gray-800' }">
            <template #header>
                <div class="flex items-center justify-between">
                    <h3 class="text-4xl">Story Book</h3>
                    <div class="flex space-x-4">
                        <UButton icon="i-heroicons-home" class="-my-1" @click="isMenuOpen = false" />
                        <UButton icon="i-heroicons-x-mark-20-solid" class="-my-1" @click="useRouter().push('/')" />
                    </div>
                </div>
            </template>
            
            <div class="flex flex-col space-y-2">
                <New class="w-full"/>
                <UButton truncate loading v-for="(_, name) in pendingStories" :key="name" size="lg" class="w-full text-xl" :label="name" icon="i-heroicons-book-open"/>
                <div class="w-full" v-for="story in stories" >
                    <UButton truncate :key="story" size="lg" class="w-5/6 text-xl" :label="story" icon="i-heroicons-book-open" @click="goToStory(story as string)"/>
                    <!-- delete icon -->
                    <UButton size="lg" variant="ghost" class="w-1/7 text-xl ml-4" icon="i-heroicons-trash" @click="deleteStory(story as string)"/>
                </div>

            </div>
        </UCard>
    </USlideover>
  </div>
</template>