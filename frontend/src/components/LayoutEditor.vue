<script setup>
import { ref, watch } from 'vue'
import { api } from '../api.js'

const props = defineProps({ store: { type: Object, required: true } })
const emit = defineEmits(['updated'])

const sections = ref([])
const newSection = ref('')
const layout = ref(null)
const busy = ref('')
const error = ref('')

async function refresh() {
  sections.value = [...(props.store.sections || [])]
  layout.value = await api.layout(props.store.id)
}
watch(() => props.store.id, refresh, { immediate: true })

async function save() {
  const s = await api.updateStore(props.store.id, { sections: sections.value })
  emit('updated', s)
  layout.value = await api.layout(props.store.id)
}

function move(i, d) {
  const j = i + d
  if (j < 0 || j >= sections.value.length) return
  const arr = sections.value
  ;[arr[i], arr[j]] = [arr[j], arr[i]]
  save()
}

function removeAt(i) {
  sections.value.splice(i, 1)
  save()
}

function add() {
  const name = newSection.value.trim()
  if (!name) return
  sections.value.push(name)
  newSection.value = ''
  save()
}

// Aisle-sign photos, selected in walking order: each photo's first OCR line
// becomes the next section.
async function importPhotos(e) {
  const files = [...e.target.files]
  error.value = ''
  for (const [i, f] of files.entries()) {
    busy.value = `Reading photo ${i + 1}/${files.length}…`
    try {
      const lines = await api.ocr(f)
      if (lines.length) sections.value.push(lines[0])
    } catch (err) {
      error.value = err.message
      break
    }
  }
  busy.value = ''
  e.target.value = ''
  save()
}

function adoptLearned() {
  sections.value = layout.value.sections.map((s) => s.name)
  save()
}
</script>

<template>
  <section>
    <h2>Known section order</h2>
    <p class="muted">
      Walk the store once and list the sections you pass — or upload photos of
      the aisle signs <em>in walking order</em>. Trips you record refine this.
    </p>
    <ol class="list numbered">
      <li v-for="(s, i) in sections" :key="s + i">
        <span class="grow">{{ s }}</span>
        <span class="actions">
          <button class="ghost" @click="move(i, -1)">↑</button>
          <button class="ghost" @click="move(i, 1)">↓</button>
          <button class="ghost" @click="removeAt(i)">✕</button>
        </span>
      </li>
    </ol>
    <form class="row" @submit.prevent="add">
      <input v-model="newSection" placeholder="Section, e.g. Bakery" />
      <button>Add</button>
    </form>
    <label class="file">
      📷 Import aisle-sign photos
      <input type="file" accept="image/*" multiple @change="importPhotos" />
    </label>
    <p v-if="busy" class="muted">{{ busy }}</p>
    <p v-if="error" class="error">{{ error }}</p>
  </section>

  <section v-if="layout">
    <h2>Generated layout</h2>
    <p class="muted">Blends your section order with what {{ store.trips }} recorded trips show.</p>
    <ol class="list numbered">
      <li v-for="s in layout.sections" :key="s.name">
        <span class="grow">
          <strong>{{ s.name }}</strong>
          <span class="badge" :class="s.source">{{ s.source }}</span>
          <div v-if="s.items.length" class="muted small">{{ s.items.join(', ') }}</div>
        </span>
      </li>
    </ol>
    <button v-if="layout.sections.length" class="ghost" @click="adoptLearned">
      Use generated order as section order
    </button>
  </section>
</template>
