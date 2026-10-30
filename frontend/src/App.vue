<script setup>
import { ref, computed, onMounted } from 'vue'
import { api, setUnauthorizedHandler } from './api.js'
import SignIn from './components/SignIn.vue'
import LayoutEditor from './components/LayoutEditor.vue'
import SortView from './components/SortView.vue'
import ReceiptView from './components/ReceiptView.vue'
import ListView from './components/ListView.vue'
import StatsView from './components/StatsView.vue'

const user = ref(null)
const checked = ref(false)
const stores = ref([])
const selectedId = ref(localStorage.getItem('storeId') || '')
const tab = ref('list')
const listId = ref(localStorage.getItem('listId') || '')
const newName = ref('')
const error = ref('')
const tabs = [
  { id: 'list', icon: '📝', label: 'List' },
  { id: 'sort', icon: '🛒', label: 'Shop' },
  { id: 'receipt', icon: '🧾', label: 'Receipt' },
  { id: 'layout', icon: '🗺️', label: 'Layout' },
  { id: 'stats', icon: '📊', label: 'Stats' },
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
}

async function remove() {
  if (!selected.value || !confirm(`Delete ${selected.value.name}?`)) return
  try {
    await api.deleteStore(selected.value.id)
    selectedId.value = ''
    await load()
  } catch {
    alert('Only the creator can delete a supermarket, and only while nobody else uses it.')
  }
}

async function signedIn(u) {
  user.value = u
  await load()
}

async function logout() {
  await api.logout()
  user.value = null
  stores.value = []
}

setUnauthorizedHandler(() => (user.value = null))

function selectList(id) {
  listId.value = id
  localStorage.setItem('listId', id)
}

function listDone() {
  selectList('')
  tab.value = 'list'
}

function onUpdated(store) {
  const i = stores.value.findIndex((s) => s.id === store.id)
  if (i >= 0) stores.value[i] = store
}

onMounted(async () => {
  try {
    user.value = await api.me()
    await load()
  } catch {
    user.value = null
  }
  checked.value = true
})
</script>

<template>
  <SignIn v-if="checked && !user" @signed-in="signedIn" />
  <template v-else-if="user">
  <header class="topbar">
    <h1>
      <span class="logo">🛒</span> <span class="grow">Shopping List Sort</span>
      <button class="avatar" :title="`Signed in as ${user.email || user.name} — sign out`" @click="logout">
        <img v-if="user.picture" :src="user.picture" alt="" referrerpolicy="no-referrer" />
        <span v-else>{{ user.name[0]?.toUpperCase() }}</span>
      </button>
    </h1>
    <div class="store-picker">
      <select :value="selectedId" aria-label="Supermarket" @change="select($event.target.value)">
        <option disabled value="">Choose a supermarket…</option>
        <option v-for="s in stores" :key="s.id" :value="s.id">
          {{ s.name }} · {{ s.trips }} {{ s.trips === 1 ? 'trip' : 'trips' }}
        </option>
      </select>
      <button v-if="selected && selected.createdBy === user.id" class="icon" title="Delete supermarket" @click="remove">🗑</button>
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
    <ListView v-if="tab === 'list'" :active-list-id="listId" @select="selectList" @shop="tab = 'sort'" />
    <SortView
      v-else-if="tab === 'sort'"
      :store="selected"
      :list-id="listId"
      @updated="onUpdated"
      @edit-list="tab = 'list'"
      @list-done="listDone"
    />
    <StatsView v-else-if="tab === 'stats'" />
    <ReceiptView v-else-if="tab === 'receipt'" :store="selected" :list-id="listId" @updated="onUpdated" @list-done="selectList('')" />
    <LayoutEditor v-else :store="selected" :user-id="user.id" @updated="onUpdated" />
  </main>
  <div v-else class="empty">
    <div class="empty-icon">🏪</div>
    <p>Add a supermarket to get started.</p>
  </div>
  </template>
</template>
