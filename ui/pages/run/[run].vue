<script lang="ts" setup>
  const route = useRoute()
  const prefs = usePrefs()

  let id = fromArray(route.params.run) as string

  definePageMeta({
    middleware: defineNuxtRouteMiddleware(async (to/*, from*/) => {
      const toId = fromArray(to.params.run) as string
      const toRun = (await useRuns().find(toId))
      if ( !toRun ) {
        window.location.href = '/'
      }
    })
  })

  const run = (await useRuns().find(id))!

  const callMap = reactive<Record<string, Call>>({})
  const referredTo = reactive<MapBool>({})

  watch(run.calls, (neu) => {
    for ( const call of neu ) {
      callMap[call.id] = call
      if ( prefs.allExpanded ) {
        prefs.expand(call.id)
      }

      for ( const msg of call.messages ) {
        if ( 'role' in msg &&  msg.role === 'assistant' && typeof msg.content !== 'string' && isArray(msg.content) ) {
          for ( const line of msg.content ) {
            if ( 'toolCall' in line ) {
              referredTo[line.toolCall.id] = true
            }
          }
        }
      }
    }
  }, {immediate: true})

  const topLevel = computed(() => {
    return run.calls.filter((x) => {
      if ( x.parentID ) {
        if ( referredTo[x.id] ) {
          return false
        }

        if ( callMap[x.parentID] ) {
          return false
        }
      }

      return true
    })
  })

  if ( !run ) {
    navigateTo('/')
  }

  function toggleAll() {
    if ( prefs.allExpanded ) {
      prefs.collapseAll()
    } else {
      prefs.expandAll(Object.keys(callMap))
    }
  }
</script>

<template>
  <div v-if="run">
    <div class="clearfix">
      <div class="float-left">
        <h1 class="text-xl">
          Run: {{id}}
        </h1>
      </div>
      <div class="float-right">
        <UButton
          size="sm"
          :icon="prefs.allExpanded ? 'i-heroicons-minus' : 'i-heroicons-plus'"
          :label="prefs.allExpanded ? 'Collapse All' : 'Expand All'"
          @click="toggleAll()"
        />

        <UBadge :color="colorForState(run.state)" size="lg" class="align-top ml-5">
          <i :class="iconForState(run.state)"/> {{ucFirst(run.state)}}
        </UBadge>
      </div>
    </div>

    <UAlert
      v-if="run?.err"
      icon="i-heroicons-exclamation-triangle"
      color="red"
      class="my-5"
      title="Error"
      variant="solid"
      :description="run.err"
    />


    <UDivider class="my-4" />

    <template v-for="c in topLevel" :key="c.id">
      <Call
        :call="c"
        :run="run"
        :referred-to="referredTo"
        :call-map="callMap"
        :depth="0"
      />
    </template>
  </div>
</template>
