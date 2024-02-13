<script lang="ts" setup>

interface Props {
  modelValue: any
}

const { modelValue } = defineProps<Props>()

const content = computed(() => {
  return `${modelValue}`.trim()
})

</script>

<template>
<span class="whitespace-pre-wrap">{{content || '<none>'}}</span>
</template>
