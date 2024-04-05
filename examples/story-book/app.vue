<script lang="ts" setup>
useHead({
  title: 'Story Book',
  meta:  [
    { charset: 'utf-8' },
    { name: 'apple-mobile-web-app-capable', content: 'yes' },
    { name: 'format-detection', content: 'telephone=no' },
    { name: 'viewport', content: `width=device-width` },
  ],
})
</script>

<template>
  <div>
    <div class="grid grid-rows-[1fc,1fr] fixed top-0 right-0 bottom-0 left-0">
      <div class="overflow-auto border-t-2 border-transparent">
        <Nav class="absolute right-5 top-5 z-10"/>
        <div class="p-5 lg:p-10 max-w-full w-full h-full mx-auto">
          <NuxtPage class="pb-10" />
        </div>
      </div>
    </div>
    <UNotifications />
  </div>
</template>
