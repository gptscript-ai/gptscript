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
    label: x.id,
    icon: iconForState(x.state), // 'i-heroicons-cog animate-spin', // iconForState(x.state),
    to: `/run/${encodeURIComponent(x.id)}`
  }})

  return sortBy(out, 'label:desc')
})

async function remove(e: MouseEvent, id: any) {
  e.stopPropagation();
  e.preventDefault();
}
</script>

<template>
  <div class="mt-5">
    <h4>GPTScripts</h4>
    <UVerticalNavigation :links="gptLinks" />

    <h4 class="mt-5">
      Runs
    </h4>
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
</template>

<style lang="scss" scoped>
  H4 {
    padding-left: 1rem;
  }

  LI {
    display: block;
  }
  .active {
    background-color: red;
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
