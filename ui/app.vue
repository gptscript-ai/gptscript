<script lang="ts" setup>
const sock = useSocket()
const router = useRouter()

useHead({
  meta: [
    { charset: 'utf-8' },
    { name: 'apple-mobile-web-app-capable', content: 'yes' },
    { name: 'format-detection', content: 'telephone=no' },
    { name: 'viewport', content: `width=device-width, height=device-height` },
  ],
})

const root = ref<HTMLDivElement>()

watch(router.currentRoute, () => {
  root.value?.classList.remove('open')
})


function toggle() {
  root.value?.classList.toggle('open')
}
</script>

<template>
  <div ref="root" class="root bg-slate-50 dark:bg-slate-950">
    <header class="flex bg-slate-300 dark:bg-slate-900">
      <div class="toggle flex-initial p-2">
        <UButton icon="i-heroicons-bars-3" @click="toggle"/>
      </div>
      <div class="flex-initial">
        <img src="~/assets/logo.svg" style="height: 40px; margin: 5px 0 5px 0.5rem;"/>
      </div>
      <div class="flex-initial">
        <img src="~/assets/logotype.svg" class="dark:invert" style="height: 30px; margin: 12px 0 8px 5px;"/>
      </div>

      <div class="flex-1"/>

      <div class="flex-initial text-right p-2" v-if="sock.sock.status !== 'OPEN'">
        <UBadge color="red" size="lg" variant="solid">
          <i class="i-heroicons-bolt-slash"/>&nbsp;{{ucFirst(sock.sock.status.toLowerCase())}}
        </UBadge>
      </div>

      <div class="text-right p-2 flex-initial">
        <ThemeToggle />
      </div>
    </header>

    <LeftNav class="left-nav bg-slate-100 dark:bg-slate-900"/>

    <main>
      <NuxtPage />
    </main>
  </div>
</template>

<style lang="scss" scoped>
  .root {
    --nav-width: 500px;
    --header-height: 50px;

    display: grid;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    overflow: hidden;

    grid-template-areas: "header header" "nav main";
    grid-template-rows: var(--header-height) 1fr;
    grid-template-columns: 300px 1fr;

    HEADER {
      grid-area: header;
    }

    .left-nav {
      grid-area: nav;
      max-height: 100vh;
    }

    MAIN {
      grid-area: main;
      overflow: auto;
      position: relative;
      padding: 1rem;
    }
  }

  // Desktop
  @media all and (min-width: 768px)  {
    .root {
      .toggle {
        display: none;
      }
    }
  }

  // Mobile
  @media all and (max-width: 767px)  {
    .root {
      grid-template-columns: 0 100%;
      transition: grid-template-columns 250ms;

      .left-nav { opacity: 0; }
      MAIN { opacity: 1}
    }
    .root.open {
      grid-template-columns: 100% 0;

      .left-nav { opacity: 1; }
      MAIN { opacity: 0; }
    }
  }

</style>
