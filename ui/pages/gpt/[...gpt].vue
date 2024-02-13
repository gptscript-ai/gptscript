<script lang="ts" setup>
  import { usePrefs } from '@/stores/prefs';
  const { metaSymbol } = useShortcuts()

  const TOOL = 'tool'

  const route = useRoute()
  const argsForm = ref()
  const prefs = usePrefs()

  let fileName = route.params.gpt.join('/')
  const gpt = await useGpts().find(fileName)

  function toolIdToName(id: string) {
    return gpt.toolSet?.[id]?.name || ''
  }

  function toolNameToId(name?: string) {
    if ( !name ) {
      return gpt.entryToolId || ''
    }

    return Object.values(gpt.toolSet||{}).find(x => x.name === name)?.id || ''
  }

  const toolName = ref(route.query[TOOL] || toolIdToName(gpt.entryToolId) || '')

  watch(toolName, (neu) => {
    navigateTo({query: {[TOOL]: neu || undefined }})
  })

  const args = ref<Args>({})
  const stringArg = ref('')

  const entryTool = computed(() => {
    return gpt.toolSet[gpt.entryToolId]
  })

  const tool = computed(() => {
    return gpt.toolSet[toolNameToId(toolName.value)]
  })

  const toolOptions = computed((): SelectOption[] => {
    const out: SelectOption[] = []

    for ( const k in entryTool.value?.localTools || {} ) {
      out.push({
        label: k || '<Default>',
        value: k
      })
    }

    return out
  })

  // -----

  async function run() {
    let input: any
    if ( tool.value.arguments ) {
      input = args.value || {}
    } else {
      input = stringArg.value || ''
    }

    const obj = await useRuns().create(fileName, toolName.value, input, prefs.cache)
    navigateTo({name: 'run-run', params: {run: obj.id}} )
  }

  defineShortcuts({
    'meta_enter': {
      usingInput: true,
      handler: () => {
        run()
      }
    }
  })
</script>

<template>
  <div v-if="tool">
    <div class="clearfix">
      <div class="float-left">
        <h1 class="text-xl py-1">
          <i class="i-heroicons-wrench-screwdriver mr-2 align-middle"/>{{fileName}}
          <span v-if="toolName">: {{toolName}}</span>
        </h1>
      </div>
      <div class="float-right mt-1">
        <USelect
          v-if="toolOptions.length > 1"
          v-model="toolName" :options="toolOptions" value-attribute="value"
        />
      </div>
    </div>

    <h2 v-if="tool.description">{{tool.description}}</h2>

    <UDivider class="my-4" />

    <Arguments
      v-if="tool.arguments"
      ref="argsForm"
      :schema="tool.arguments"
      v-model="args"
      class="mb-2"
    />
    <UTextarea
      v-else
      :autofocus="true"
      name="stringArg"
      v-model="stringArg"
      placeholder="Optional free-form input"
      class="mb-2"
    />

    <div class="clearfix">
      <div class="float-left mt-2">
        <UTooltip>
          <template #text>
            {{metaSymbol}} + Enter
          </template>

          <UButton icon="i-heroicons-play-circle" @click="run" :disabled="argsForm && !argsForm.valid">
            Run
          </UButton>
        </UTooltip>
      </div>
      <div class="float-right">
        <UFormGroup label="Cache" size="xs">
          <UToggle v-model="prefs.cache"/>
        </UFormGroup>
      </div>
    </div>
  </div>
  <div v-else>
    <UAlert
      icon="i-heroicons-exclamation-triangle"
      color="red"
      class="my-5"
      title="Error"
      variant="solid"
      description="Tool not found"
    />
  </div>
</template>
