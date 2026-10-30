<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from './api.js'
import LayoutEditor from './components/LayoutEditor.vue'
import SortView from './components/SortView.vue'
import ReceiptView from './components/ReceiptView.vue'

const stores = ref([])
const selectedId = ref(localStorage.getItem('storeId') || '')
const tab = ref('sort')
const newName = ref('')
const error = ref('')
const tabs = [
  { id: 'sort', icon: '📝', label: 'List' },
  { id: 'receipt', icon: '🧾', label: 'Receipt' },
  { id: 'layout', icon: '🗺️', label: 'Layout' },
]

const selected = computed(() => stores.value.find((s) => s.id === selectedId.value))

async function load() {
  try {
    stores.value = (await api.stores()).sort((a, b) => a.name.localeCompare(b.name))
    if (!selected.value && stores.value.length) select(stores.value[0].id)
  } catch (e) {
    error.value = e.message
  }
}

function select(id) {
  selectedId.value = id
  localStorage.setItem('storeId', id)
}

async function create() {
  if (!newName.value.trim()) return
  const s = await api.createStore(newName.value.trim(), [])
  newName.value = ''
  await load()
  document.querySelector('.add-store')?.removeAttribute('open')
  select(s.id)
  tab.value = 'layout'
}

async function remove() {
  if (!selected.value || !confirm(`Delete ${selected.value.name}?`)) return
  await api.deleteStore(selected.value.id)
  selectedId.value = ''
  await load()
}

function onUpdated(store) {
  const i = stores.value.findIndex((s) => s.id === store.id)
  if (i >= 0) stores.value[i] = store
}

onMounted(load)
</script>

<template>
  <header class="topbar">
    <h1><span class="logo">🛒</span> Shopping List Sort</h1>
    <div class="store-picker">
      <select :value="selectedId" aria-label="Supermarket" @change="select($event.target.value)">
        <option disabled value="">Choose a supermarket…</option>
        <option v-for="s in stores" :key="s.id" :value="s.id">
          {{ s.name }} · {{ s.trips }} trips
        </option>
      </select>
      <button v-if="selected" class="icon" title="Delete supermarket" @click="remove">🗑</button>
    </div>
    <details class="add-store" :open="!stores.length">
      <summary>+ Add supermarket</summary>
      <form class="row" @submit.prevent="create">
        <input v-model="newName" placeholder="e.g. Albert Heijn Centrum" />
        <button>Add</button>
      </form>
    </details>
    <p v-if="error" class="error">{{ error }}</p>
  </header>

  <main v-if="selected">
    <nav class="tabs">
      <button v-for="t in tabs" :key="t.id" :class="{ active: tab === t.id }" @click="tab = t.id">
        <span class="tab-icon">{{ t.icon }}</span>
        <span>{{ t.label }}</span>
      </button>
    </nav>
    <SortView v-if="tab === 'sort'" :store="selected" @updated="onUpdated" />
    <ReceiptView v-else-if="tab === 'receipt'" :store="selected" @updated="onUpdated" />
    <LayoutEditor v-else :store="selected" @updated="onUpdated" />
  </main>
  <div v-else class="empty">
    <div class="empty-icon">🏪</div>
    <p>Add a supermarket to get started.</p>
  </div>
</template>
