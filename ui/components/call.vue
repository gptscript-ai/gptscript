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

const children = computed(() => {
  return run.calls.filter(x => !referredTo[x.id] && x.parentID && x.parentID === call.id)
})

const isExpanded = computed(() => {
  return prefs.allExpanded ||  prefs.expanded[call.id] || false
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
</script>
<template>
  <UCard :key="call.id" class="call">
    <i v-if="prefs.debug" class="text-blue-400">{{call}}</i>
    <div class="clearfix">
      <div class="float-left">
        <div>
          <UButton v-if="children.length || call.messages?.length" size="xs" :icon="icon" @click="toggle" class="align-baseline"/>
          {{displayName}}
        </div>
      </div>
      <div class="float-right">
        <UBadge class="align-baseline mr-2" :id="call.id">
          <i class="i-heroicons-key"/>&nbsp;{{ prefs.mapCall(call.id) }}
        </UBadge>
        <UBadge :color="colorForState(call.state)" class="align-baseline">
          <i :class="iconForState(call.state)"/>&nbsp;{{ucFirst(call.state)}}
        </UBadge>
      </div>
    </div>
    <div>
      <b>Input:</b> {{call.input || '<none>'}}
    </div>
    <div>
      <b>Output:</b> <span v-html="nlToBr(call.output || '<none>')"/>
    </div>

    <template v-if="isExpanded && call.messages?.length">
      <Message
        v-for="(msg, idx) in call.messages"
        :key="idx"
        class="my-2"
        :run="run"
        :call="call"
        :msg="msg"
        :referred-to="referredTo"
        :call-map="callMap"
        :depth="depth+1"
      />
    </template>

    <div v-if="isExpanded && children.length" class="ml-5">
      CHILDREN
      {{referredTo}}
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
