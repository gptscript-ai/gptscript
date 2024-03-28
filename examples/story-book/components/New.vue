<script setup lang="ts">
import { useMainStore } from '@/store'

const store = useMainStore()
const open = ref(false)
const pageOptions = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
const toast = useToast()
const state = reactive({
    prompt: '',
    pages: 0,
})

console.log(Object.keys(store.pendingStories))

async function onSubmit () {
    const response = await fetch('/api/story', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(state)
    })

    if (response.ok) {
        toast.add({
            id: 'story-generating',
            title: 'Story Generating',
            description: 'Your story is being generated. Depending on the length, this may take a few minutes.',
            icon: 'i-heroicons-pencil-square-solid',
        })

        const es = new EventSource(`/api/story/sse?prompt=${state.prompt}`)
        es.onmessage = async (event) => {
            store.addPendingStoryMessage(state.prompt, event.data)
            if ((event.data as string) === 'done') {
                es.close()
                store.removePendingStory(state.prompt)
                store.fetchStories()
                toast.add({
                    id: 'story-created',
                    title: 'Story Created',
                    description: 'A story you requested has been created.',
                    icon: 'i-heroicons-check-circle-solid',
                })
            } else if ((event.data as string).includes('error')) {
                es.close()
                store.removePendingStory(state.prompt)

                toast.add({
                    id: 'story-generating-failed',
                    title: 'Story Generation Failed',
                    description: `Your story could not be generated due to an error.\n\n${event.data}.`,
                    icon: 'i-heroicons-x-mark',
                    timeout: 30000,
                })
            }
        }

        if (state.prompt.length){
            store.addPendingStory(state.prompt, es)
        }
        open.value = false
    } else {
        toast.add({
            id: 'story-generating-failed',
            title: 'Story Generation Failed',
            description: `Your story could not be generated due to an error: ${response.statusText}.`,
            icon: 'i-heroicons-x-mark',
        })
    }
}
</script>


<template>
    <UPopover v-if="Object.keys(store.pendingStories).length === 0" v-model:open="open" class="overflow-visible">
        <UButton size="lg" class="w-full text-xl" icon="i-heroicons-plus">New Story</UButton>
        <template #panel>
            <UForm class="w-[15vw] p-8" :state=state @submit="onSubmit">
                <UFormGroup label="Prompt" name="prompt">
                    <UTextarea v-model="state.prompt" label="Prompt"/>
                </UFormGroup>
                <UFormGroup class="my-6" label="Pages" name="pages">
                    <USelect class="overflow-visible"v-model="state.pages" :options="pageOptions" />
                </UFormGroup>
                <UButton class="w-full" type="submit">Create Story</UButton>
            </UForm>
        </template>
    </UPopover>
</template>