<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n.js'

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
    <h2>{{ t('habits') }}</h2>
    <div class="row">
      <select v-model="templateId" class="wide-select">
        <option value="">{{ t('allLists') }}</option>
        <option v-for="tp in templates" :key="tp.id" :value="tp.id">{{ t('templatePrefix', tp.name) }}</option>
      </select>
      <select v-model.number="window">
        <option :value="4">{{ t('lastN', 4) }}</option>
        <option :value="8">{{ t('lastN', 8) }}</option>
        <option :value="0">{{ t('allTime') }}</option>
      </select>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="report && !report.lists" class="muted">{{ t('noCompleted') }}</p>
  </section>

  <section v-if="template && hasSuggestions">
    <h2>{{ t('suggestions', template.name) }}</h2>
    <ul class="list">
      <li v-for="s in report.addToTemplate" :key="'a' + s.name">
        <span class="grow"><strong>{{ s.name }}</strong><div class="muted small">{{ s.reason }}</div></span>
        <button @click="apply('add', s)">{{ t('add') }}</button>
      </li>
      <li v-for="s in report.adjustQty" :key="'q' + s.name">
        <span class="grow"><strong>{{ s.name }}</strong><div class="muted small">{{ s.reason }}</div></span>
        <button @click="apply('qty', s)">{{ t('set', s.qty) }}</button>
      </li>
      <li v-for="s in report.removeFromTemplate" :key="'r' + s.name">
        <span class="grow"><strong>{{ s.name }}</strong><div class="muted small">{{ s.reason }}</div></span>
        <button class="ghost" @click="apply('remove', s)">{{ t('remove') }}</button>
      </li>
    </ul>
  </section>

  <section v-if="report && report.lists">
    <h2>Items <small class="muted">{{ t('lists', report.lists) }}</small></h2>
    <ul class="list">
      <li v-for="it in report.items" :key="it.name">
        <span class="grow">
          <strong>{{ it.name }}</strong>
          <span v-if="template && !it.inTemplate" class="badge">{{ t('notInTemplate') }}</span>
          <div class="bar"><div :style="{ width: pct(it.frequency) }" /></div>
        </span>
        <span class="stat">
          <strong>{{ pct(it.frequency) }}</strong>
          <span class="muted small">× {{ it.avgQty.toFixed(1) }}</span>
        </span>
      </li>
    </ul>
    <p class="muted small">{{ t('statsFoot') }}</p>
  </section>
</template>
