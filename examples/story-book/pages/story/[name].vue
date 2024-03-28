<script setup lang="ts">
    import type { Pages } from '@/lib/types'
    
    const route = useRoute()

    const name = route.params.name as string
    const pages = ref<Pages>({})
    const currentPage = ref(1)

    onMounted(async () => pages.value = await $fetch(`/api/story/${name}`) as Pages )

    const nextPage = () => {
        currentPage.value++
    }

    const prevPage = () => {
        currentPage.value--
    }

</script>

<template>
    <div class="text-2xl w-full">
        <div class="flex justify-center">
            <div class="absolute w-full bg-green-200 dark:bg-zinc-950 pt-10 top-0">
                <div class="text-center">
                    <h1 class="text-4xl font-semibold">{{ name }}</h1>
                    <h2 class="text-2xl font-thin">Page {{ currentPage }}</h2>
                </div>
                <UDivider class="mt-10" />
            </div>
        </div>
        <div class="h-full">
            <UButton v-if="currentPage > 1" @click="prevPage"
                class="absolute left-[4vw] top-1/2"
                :ui="{ rounded: 'rounded-full' }" 
                icon="i-heroicons-arrow-left" 
            />
            <UButton v-if="currentPage < Object.keys(pages).length" @click="nextPage"
                class="absolute right-[4vw] top-1/2"
                :ui="{ rounded: 'rounded-full' }"
                icon="i-heroicons-arrow-right"
            />
        </div>
        <div v-if="Object.keys(pages).length > 0" class="display flex-col xl:flex-row flex content-center justify-center space-y-20 xl:space-x-20 xl:space-y-0 items-center mx-16 mt-44">
            <p class="xl:w-1/2 text-2xl">{{ pages[currentPage].content }}</p>
            <img class="xl:w-2/5" :src="pages[currentPage].image_path" alt="page image" />
        </div>
    </div>
</template>