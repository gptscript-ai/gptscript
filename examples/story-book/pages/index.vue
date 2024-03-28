<script setup lang="ts">
    import { useMainStore } from '@/store'
    const store = useMainStore()
    onMounted(async () => store.fetchStories() )
</script>

<template>
    <div class="w-full h-full flex items-center justify-center">
        <div class="w-1/4">
            <h1 class="text-4xl text-center">Welcome to your story book!</h1>
            <UDivider class="w-full my-10">Your stories</UDivider>  
            <Stories />
        </div>
    </div>
</template>