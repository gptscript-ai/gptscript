<script setup lang="ts">
    import type { Pages } from '@/lib/types'
    import unmangleStoryName from '@/lib/unmangle';
    import jsPDF from 'jspdf'

    const route = useRoute()

    const name = route.params.name as string
    const pages = ref<Pages>({})
    const currentPage = ref(1)

    onMounted(async () => pages.value = await $fetch(`/api/story/${name}`) as Pages )
    const nextPage = () => currentPage.value++
    const prevPage = () => currentPage.value--

    const exportToPDF = () => {
        const doc = new jsPDF()

        // Add the title page
        doc.text(
            unmangleStoryName(name), 
            (doc.internal.pageSize.getWidth() - 10) / 2 , 
            (doc.internal.pageSize.getHeight() - 10) / 2,
            { align: 'center' }
        )
        doc.addPage()

        // Add the content pages
        Object.keys(pages.value).forEach((page) => {
            // Split the text into chunks of 400 words
            const text = pages.value[page].content
            const words = text.split(' ')
            const chunks = []
            let chunk = ''
            for (let i = 0; i < words.length; i++) {
                chunk += words[i] + ' '
                if ((i + 1) % 400 === 0 || i === words.length - 1) {
                    chunks.push(chunk.trim())
                    chunk = ''
                }
            }

            // Add the text to the PDF
            chunks.forEach((chunk) => {
                const textLines = doc.splitTextToSize(chunk, 180) // Adjust the width as needed
                const textHeight = doc.getTextDimensions(textLines).h
                const y = (doc.internal.pageSize.getHeight() - textHeight) / 2
                doc.text(textLines, 20, y)
                doc.addPage()
            })

            // Add the image to the PDF
            doc.addImage(pages.value[page].image_path, 'JPEG', 10, (doc.internal.pageSize.getHeight() - 190) / 2, 190, 190)
            doc.addPage()
        })

        doc.deletePage(doc.internal.pages.length - 1) // a small hack to remove the last empty page that gets added
        doc.save(`${name}.pdf`)
    }
</script>

<template>
    <div class="page text-2xl w-full">
        <div class="flex justify-center">
            <div class="absolute w-full bg-green-200 dark:bg-zinc-950 pt-20 top-0">
                <div class="text-center">
                    <h1 class="text-4xl font-semibold">{{ unmangleStoryName(name) }}</h1>
                    <h2 class="text-2xl font-thin">Page {{ currentPage }}</h2>
                </div>
                <UDivider class="mt-10" />
            </div>
        </div>
        <div class="h-full flex flex-col items-center justify-center">
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
            <UButton @click="exportToPDF"
                class="absolute right-[4vw] bottom-[4vh]"
                :ui="{ rounded: 'rounded-full' }"
                icon="i-heroicons-arrow-down-on-square"
            >Export to PDF</UButton>
        </div>
        <div v-if="Object.keys(pages).length > 0" class="story flex-col xl:flex-row flex content-center justify-center space-y-20 xl:space-x-20 xl:space-y-0 items-center mx-16 mt-52">
            <p class="xl:w-1/2 text-2xl">{{ pages[currentPage].content }}</p>
            <img class="xl:w-2/5" :src="pages[currentPage].image_path" alt="page image" />
        </div>
    </div>
</template>