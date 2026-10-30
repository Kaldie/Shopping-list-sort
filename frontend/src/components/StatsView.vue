<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { api } from '../api.js'

const templates = ref([])
const templateId = ref('')
const window = ref(8)
const report = ref(null)
const error = ref('')

const template = computed(() => templates.value.find((t) => t.id === templateId.value))
const hasSuggestions = computed(
  () =>
    report.value &&
    report.value.addToTemplate.length + report.value.removeFromTemplate.length + report.value.adjustQty.length > 0,
)

async function load() {
  error.value = ''
  try {
    report.value = await api.stats(templateId.value, window.value)
  } catch (e) {
    error.value = e.message
  }
}

onMounted(async () => {
  templates.value = await api.templates()
  if (templates.value.length) templateId.value = templates.value[0].id
  else load()
})
watch([templateId, window], load)

// Apply a suggestion to the template, then refresh the report.
async function apply(kind, s) {
  const key = s.name.toLowerCase()
  let items = template.value.items
  if (kind === 'add') items = [...items, { name: s.name, qty: s.qty }]
  if (kind === 'remove') items = items.filter((i) => i.name.toLowerCase() !== key)
  if (kind === 'qty') items = items.map((i) => (i.name.toLowerCase() === key ? { ...i, qty: s.qty } : i))
  const t = await api.updateTemplate(template.value.id, { items })
  templates.value = templates.value.map((x) => (x.id === t.id ? t : x))
  await load()
}

const pct = (f) => `${Math.round(f * 100)}%`
</script>

<template>
  <section>
    <h2>Buying habits</h2>
    <div class="row">
      <select v-model="templateId" aria-label="Template" class="wide-select">
        <option value="">All lists</option>
        <option v-for="t in templates" :key="t.id" :value="t.id">Template: {{ t.name }}</option>
      </select>
      <select v-model.number="window" aria-label="Period">
        <option :value="4">Last 4 lists</option>
        <option :value="8">Last 8 lists</option>
        <option :value="0">All time</option>
      </select>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="report && !report.lists" class="muted">
      No completed lists yet. Finish a shopping trip or tap “Mark as bought” on a list.
    </p>
  </section>

  <section v-if="template && hasSuggestions">
    <h2>Suggestions for “{{ template.name }}”</h2>
    <ul class="list">
      <li v-for="s in report.addToTemplate" :key="'a' + s.name">
        <span class="grow"><strong>{{ s.name }}</strong><div class="muted small">{{ s.reason }}</div></span>
        <button @click="apply('add', s)">Add</button>
      </li>
      <li v-for="s in report.adjustQty" :key="'q' + s.name">
        <span class="grow"><strong>{{ s.name }}</strong><div class="muted small">{{ s.reason }}</div></span>
        <button @click="apply('qty', s)">Set {{ s.qty }}</button>
      </li>
      <li v-for="s in report.removeFromTemplate" :key="'r' + s.name">
        <span class="grow"><strong>{{ s.name }}</strong><div class="muted small">{{ s.reason }}</div></span>
        <button class="ghost" @click="apply('remove', s)">Remove</button>
      </li>
    </ul>
  </section>

  <section v-if="report && report.lists">
    <h2>Items <small class="muted">{{ report.lists }} lists</small></h2>
    <ul class="list">
      <li v-for="it in report.items" :key="it.name">
        <span class="grow">
          <strong>{{ it.name }}</strong>
          <span v-if="template && !it.inTemplate" class="badge">not in template</span>
          <div class="bar"><div :style="{ width: pct(it.frequency) }" /></div>
        </span>
        <span class="stat">
          <strong>{{ pct(it.frequency) }}</strong>
          <span class="muted small">× {{ it.avgQty.toFixed(1) }}</span>
        </span>
      </li>
    </ul>
    <p class="muted small">% of lists the item was bought in · average quantity when bought.</p>
  </section>
</template>
