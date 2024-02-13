<script lang="ts" setup>
// const router = useRouter()
const gptList = await useGpts().listAll()
const runList = await useRuns().findAll()

const gptLinks = computed(() => {
  return (gptList || []).map(x => { return {
    label: x,
    icon: 'i-heroicons-wrench-screwdriver',
    to: `/gpt/` + x.split('/').map(y => encodeURIComponent(y)).join('/'),
    }})
})

const runLinks = computed(() => {
  const out = (runList || []).map(x => { return {
    id: x.id,
    label: (x.program?.name || '') + ' #' + x.id,
    icon: iconForState(x.state), // 'i-heroicons-cog animate-spin', // iconForState(x.state),
    to: `/run/${encodeURIComponent(x.id)}`
  }})

  return sortBy(out, 'id:desc')
})

async function remove(e: MouseEvent, id: any) {
  e.stopPropagation();
  e.preventDefault();
}
</script>

<template>
  <nav class="left">
    <div class="scripts text-slate-700 dark:text-slate-400">
      <h4 class="header px-3 py-2 bg-slate-200 dark:bg-slate-800">Scripts</h4>
      <UVerticalNavigation :links="gptLinks" />
    </div>

    <div class="runs text-slate-700 dark:text-slate-400">
      <h4 class="header px-3 py-2 bg-slate-200 dark:bg-slate-800">Run History</h4>
      <UVerticalNavigation :links="runLinks">
        <template #badge="{ link }">
          <UButton
            class="absolute right-2 delete-btn"
            icon="i-heroicons-trash"
            aria-label="Delete"
            @click="e => remove(e, link.id)"
            size="xs"
          />
        </template>
      </UVerticalNavigation>
    </div>
  </nav>
</template>

<style lang="scss" scoped>
  .left {
    display: grid;
    grid-template-areas: "scripts" "runs";
    grid-template-rows: 1fr 1fr;
    height: 100%;
  }

  .scripts {
    grid-area: scripts;
    overflow-y: auto;
  }

  .runs {
    grid-area: runs;
    overflow-y: auto;
  }

  .delete-btn {
    display: none;
  }

/*
  :deep(A:hover .delete-btn) {
    display: initial;
  }
*/
</style>
