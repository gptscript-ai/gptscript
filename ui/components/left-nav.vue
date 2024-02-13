<script lang="ts" setup>
// const router = useRouter()
const gptList = await useGpts().listAll()
const runList = await useRuns().findAll()
const sock = useSocket()

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
      <h4 class="header px-3 py-2 bg-slate-200 dark:bg-slate-800">GPTScripts</h4>
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

    <aside class="px-2 flex">
      <div class="flex-1 mt-1">
        <ThemeToggle />
      </div>
      <div class="flex-initial text-right" v-if="sock.sock.status !== 'OPEN'">
        <UBadge color="red" class="mt-2">
          <i class="i-heroicons-bolt-slash"/>&nbsp;{{ucFirst(sock.sock.status.toLowerCase())}}
        </UBadge>
      </div>
    </aside>
  </nav>
</template>

<style lang="scss" scoped>
  $aside-height: 45px;

  .left {
    display: grid;
    grid-template-areas: "scripts" "runs" "aside";
    grid-template-rows: 1fr 1fr $aside-height;
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

  ASIDE {
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
