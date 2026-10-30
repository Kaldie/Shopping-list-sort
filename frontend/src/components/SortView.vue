<script setup>
import { ref, computed, watch } from 'vue'
import { api, toLines } from '../api.js'

const props = defineProps({ store: { type: Object, required: true } })
const emit = defineEmits(['updated'])

const text = ref(localStorage.getItem('list') || '')
const groups = ref([])
const picked = ref([]) // items in the order they were checked off
const busy = ref('')
const error = ref('')

watch(text, (t) => localStorage.setItem('list', t))
watch(() => props.store.id, () => { groups.value = []; picked.value = [] })

const total = computed(() => groups.value.reduce((n, g) => n + g.items.length, 0))

async function sort() {
  error.value = ''
  try {
    groups.value = (await api.sort(props.store.id, toLines(text.value))).groups
    picked.value = []
  } catch (e) {
    error.value = e.message
  }
}

async function fromPhoto(e) {
  const f = e.target.files[0]
  if (!f) return
  busy.value = 'Reading list…'
  error.value = ''
  try {
    const lines = await api.ocr(f)
    text.value = [text.value, ...lines].filter(Boolean).join('\n')
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = ''
    e.target.value = ''
  }
}

function toggle(item) {
  const i = picked.value.indexOf(item)
  if (i >= 0) picked.value.splice(i, 1)
  else picked.value.push(item)
}

// The check-off order is an observed walk through the store: send it as a trip.
async function finish() {
  if (picked.value.length < 2) {
    error.value = 'Check off at least two items to record a trip.'
    return
  }
  const s = await api.trip(props.store.id, picked.value)
  emit('updated', s)
  const remaining = toLines(text.value).filter(
    (l) => !picked.value.some((p) => p.toLowerCase() === l.toLowerCase()),
  )
  text.value = remaining.join('\n')
  groups.value = []
  picked.value = []
}
</script>

<template>
  <section>
    <h2>Shopping list</h2>
    <textarea v-model="text" rows="8" placeholder="One item per line" />
    <div class="row">
      <button @click="sort">Sort for {{ store.name }}</button>
      <label class="file">
        📷 From photo
        <input type="file" accept="image/*" capture="environment" @change="fromPhoto" />
      </label>
    </div>
    <p v-if="busy" class="muted">{{ busy }}</p>
    <p v-if="error" class="error">{{ error }}</p>
  </section>

  <section v-if="groups.length">
    <h2>Route <small class="muted">{{ picked.length }}/{{ total }}</small></h2>
    <p class="muted small">
      Tick items off as you put them in your cart — the order you tick them
      teaches the app this store's layout.
    </p>
    <div v-for="g in groups" :key="g.section" class="group">
      <h3>{{ g.section }}</h3>
      <label v-for="item in g.items" :key="item" class="check" :class="{ done: picked.includes(item) }">
        <input type="checkbox" :checked="picked.includes(item)" @change="toggle(item)" />
        {{ item }}
      </label>
    </div>
    <button @click="finish">Finish trip &amp; learn</button>
  </section>
</template>
