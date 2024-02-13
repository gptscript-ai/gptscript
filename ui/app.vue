<script lang="ts" setup>
  import '@/styles/app.scss'
</script>
<template>
  <div class="root bg-slate-100 dark:bg-slate-950">
    <LeftNav class="left-nav bg-slate-200 dark:bg-slate-900"/>
    <main>
      <NuxtPage />
    </main>
  </div>
</template>

<style lang="scss" scoped>
  .root {
    --nav-width: 300px;

    display: grid;
    grid-template-areas: "nav main";
    grid-template-columns: var(--nav-width) 1fr;

    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;

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
</style>
