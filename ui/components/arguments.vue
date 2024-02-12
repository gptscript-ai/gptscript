<script lang="ts" setup>
interface Props {
  modelValue?: Args
  schema: ArgSchema
}

const { schema, modelValue: initialState } = defineProps<Props>()
const form = ref()
const state = reactive(clone(initialState || {}))

if ( schema ) {
  for ( const k in schema.properties ) {
    if ( typeof state[k] !== 'string' ) {
      state[k] = ''
    }
  }
}

// function validate(state: Args): FormError[] {
//   const errors: FormError[] = []
function validate(state: Args) {
  const errors: {path: string, message: string}[] = []

  if ( schema && schema.required?.length ) {
    for ( const k of schema.required ) {
      if ( ! (state[k] || '').trim() ) {
        errors.push({ path: k, message: 'Required' })
      }
    }
  }

  return errors
}

const valid = computed(() => {
  return form.value?.errors?.length === 0 || false
})

defineExpose({form, valid})

// async function onError (event: FormErrorEvent) {
async function onError (event: any) {
  const element = document.getElementById(event.errors[0].id)
  element?.focus()
  element?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

// eslint-disable-next-line func-call-spacing
const emit = defineEmits<{
  (e: 'update:modelValue', value: Args): void
}>()

watchEffect(() => {
  emit('update:modelValue', state)
})
</script>

<template>
  <div>
    <UForm
      ref="form"
      :validate="validate"
      :state="state"
      class="space-y-4"
      @error="onError"
    >
      <template v-if="schema">
        <UFormGroup v-for="(p, k) in schema?.properties" :key="k" :label="k" :name="k" :required="schema.required?.includes(k)">
          <UTextarea autoresize :rows="1" :value="state[k]" @update:modelValue="(v) => state[k] = v" />
        </UFormGroup>
      </template>
      <template v-else>
          <UTextarea autoresize :rows="1" :value="state[k]" @update:modelValue="(v) => state[k] = v" />
      </template>
    </UForm>
  </div>
</template>
