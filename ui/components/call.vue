<script setup lang="ts">

interface Props {
  run: Run
  call: Call
  referredTo: MapBool
  callMap: Record<string,Call>
  depth?: number
}

const { run, call, referredTo, callMap, depth=0 } = defineProps<Props>()
const prefs = usePrefs()

const isExpanded = computed(() => {
  return prefs.allExpanded ||  prefs.expanded[call.id] || false
})

const children = computed(() => {
  return run.calls.filter(x => (!isExpanded.value || !call.showSystemMessages || !referredTo[x.id]) && x.parentID && x.parentID === call.id)
})

const messages = computed(() => {
  const all = call.messages || []

  if ( call.showSystemMessages ) {
    return all.slice()
  } else {
    return all.filter((msg) => {
      if ( ('role' in msg) ) {
        return false
      }

      return true
    })
  }
})

const icon = computed(() => {
  if ( isExpanded.value ) {
    return 'i-heroicons-minus'
  } else {
    return 'i-heroicons-plus'
  }
})

function toggle() {
  if ( isExpanded.value ) {
    prefs.collapse(call.id)
  } else {
    prefs.expand(call.id)
  }
}

const displayName = computed(() => {
  const id = call.tool?.id
  if ( !id ) {
    return call.id
  }

  const tool = run.program?.toolSet?.[id]
  if ( tool ) {
    return tool.name || tool.id.replace(/:1$/,'')
  }

  for ( const k in call.tool?.toolMapping || {} ) {
    if ( call.tool?.toolMapping[k] === id ) {
      return k
    }
  }

  return id.replace(/:1$/,'')
})

const inputLimit = 50
const outputLimit = 100
const inputDisplay = computed(() => {
  return `${call.input || '<none>'}`
})

const inputShort = computed(() => {
  return inputDisplay.value.substring(0, inputLimit) + (inputTruncated.value ? '…' : '')
})

const inputTruncated = computed(() => {
  return inputDisplay.value.length > inputLimit
})

const outputDisplay = computed(() => {
  return `${call.output || '<none>'}`
})

const outputShort = computed(() => {
  return outputDisplay.value.substring(0, outputLimit) + (outputTruncated.value ? '…' : '')
})

const outputTruncated = computed(() => {
  return outputDisplay.value.length > outputLimit
})

</script>
<template>
  <UCard :key="call.id" class="call mt-3">
    <i v-if="prefs.debug" class="text-blue-400">{{call}}</i>
    <div class="flex" :class="[isExpanded && 'mb-2']">
      <div class="flex-1">
        <div>
          <UButton v-if="children.length || call.messages?.length || inputTruncated || outputTruncated" size="xs" :icon="icon" @click="toggle" class="align-baseline mr-1"/>
          {{displayName}}
          <template v-if="!isExpanded">({{inputShort}}) <i class="i-heroicons-arrow-right align-text-bottom"/> {{outputShort}}</template>
        </div>
      </div>
      <div class="flex-inline text-right">
        <UTooltip v-if="isExpanded" :text="call.showSystemMessages ? 'Internal messages shown' : 'Internal messages hidden'">
          <UToggle v-model="call.showSystemMessages" class="mr-2" off-icon="i-heroicons-bars-2" on-icon="i-heroicons-bars-4" />
        </UTooltip>

        <!-- <UBadge size="md" class="align-baseline mr-2" :id="call.id" variant="subtle">
          <i class="i-heroicons-key"/>&nbsp;{{ prefs.mapCall(call.id) }}
        </UBadge> -->
        <UBadge size="md" :color="colorForState(call.state)" class="align-baseline" variant="subtle">
          <i :class="iconForState(call.state)"/>&nbsp;{{ucFirst(call.state)}}
        </UBadge>
      </div>
    </div>

    <div v-if="isExpanded">
      <b>Input:</b> <span class="whitespace-pre-wrap">{{call.input || '<none>'}}</span>
    </div>
    <div v-if="isExpanded">
      <b>Output:</b> <span class="whitespace-pre-wrap">{{call.output || '<none>'}}</span>
    </div>

    <template v-if="isExpanded">
      <Message
        v-for="(msg, idx) in messages"
        :key="idx"
        class="mt-1"
        :run="run"
        :call="call"
        :msg="msg"
        :referred-to="referredTo"
        :call-map="callMap"
        :depth="depth+1"
      />
    </template>

    <div v-if="children.length" class="ml-9">
      <Call
        v-for="(child, idx) in children"
        :key="idx"
        :run="run"
        :call="child"
        :referred-to="referredTo"
        :call-map="callMap"
        :depth="depth+1"
      />
    </div>
  </UCard>
</template>
