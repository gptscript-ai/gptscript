<script setup lang="ts">
const isMenuOpen = ref(false)
</script>

<template>
  <div>
    <div class="flex space-x-4">
        <DisplayMode />
        <UButton icon="i-heroicons-home" @click="() => { useRouter().push('/'); isMenuOpen = false }" />
        <UButton icon="i-heroicons-bars-3" @click="isMenuOpen = true" />
    </div>

    <USlideover v-model="isMenuOpen">
        <UCard class="flex flex-col flex-1" :ui="{ body: { base: 'flex-1' }, ring: '', divide: 'divide-y divide-gray-100 dark:divide-gray-800' }">
            <template #header>
                <div class="flex items-center justify-between">
                    <h3 class="text-4xl">Story Book</h3>
                    <div class="flex space-x-4">
                        <UButton icon="i-heroicons-x-mark-20-solid" @click="isMenuOpen = false" />
                    </div>
                </div>
            </template>
            <Stories />
        </UCard>
    </USlideover>
  </div>
</template>