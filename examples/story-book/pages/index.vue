<script setup lang="ts">
    import { useMainStore } from '@/store'

    const store = useMainStore()
    const stories = computed(() => store.stories)
    const pendingStories = computed(() => store.pendingStories)

    onMounted(async () => store.fetchStories() )
    const goToStory = (name: string) => useRouter().push(`/story/${name}`)
</script>

<template>
    <div class="w-full h-full flex items-center justify-center">
        <div>
            <h1 class="text-4xl">Welcome to your story book!</h1>
            <UDivider class="w-full my-10">Your stories</UDivider>  
            <div class="w-full flex flex-col space-y-4 max-h-[50vh] overflow-y-scroll">
                <New class="w-full"/>
                <UButton truncate size="xl" class="text-xl" icon="i-heroicons-trash" loading v-for="(_, story) in pendingStories">{{ story }}</UButton>
                <UButton truncate size="xl" class="text-xl" icon="i-heroicons-book-open" v-for="story in stories" @click="goToStory(story)">{{ story }}</UButton>
            </div>
        </div>
    </div>
</template>