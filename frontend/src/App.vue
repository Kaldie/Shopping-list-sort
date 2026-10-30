<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from './api.js'
import LayoutEditor from './components/LayoutEditor.vue'
import SortView from './components/SortView.vue'

const stores = ref([])
const selectedId = ref(localStorage.getItem('storeId') || '')
const tab = ref('sort')
const newName = ref('')
const error = ref('')

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
  <header>
    <h1>🛒 Shopping List Sort</h1>
    <div class="row">
      <select :value="selectedId" @change="select($event.target.value)">
        <option disabled value="">Choose a supermarket…</option>
        <option v-for="s in stores" :key="s.id" :value="s.id">
          {{ s.name }} ({{ s.trips }} trips)
        </option>
      </select>
      <button v-if="selected" class="ghost" @click="remove">Delete</button>
    </div>
    <form class="row" @submit.prevent="create">
      <input v-model="newName" placeholder="New supermarket name" />
      <button>Add</button>
    </form>
    <p v-if="error" class="error">{{ error }}</p>
  </header>

  <main v-if="selected">
    <nav class="tabs">
      <button :class="{ active: tab === 'sort' }" @click="tab = 'sort'">Sort &amp; shop</button>
      <button :class="{ active: tab === 'layout' }" @click="tab = 'layout'">Layout</button>
    </nav>
    <SortView v-if="tab === 'sort'" :store="selected" @updated="onUpdated" />
    <LayoutEditor v-else :store="selected" @updated="onUpdated" />
  </main>
  <p v-else class="muted">Add a supermarket to get started.</p>
</template>
