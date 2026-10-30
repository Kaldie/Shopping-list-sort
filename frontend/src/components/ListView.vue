<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import ItemsEditor from './ItemsEditor.vue'

// activeListId is the list the Shop tab sorts.
const props = defineProps({ activeListId: { type: String, default: '' } })
const emit = defineEmits(['select', 'shop'])

const lists = ref([])
const templates = ref([])
const suggestions = ref([])
const editingTemplate = ref(null) // template being edited, or null
const error = ref('')

const openLists = computed(() => lists.value.filter((l) => !l.completedAt))
const active = computed(() => lists.value.find((l) => l.id === props.activeListId))
const fromTemplate = computed(() => templates.value.find((t) => t.id === active.value?.templateId))

async function load() {
  try {
    ;[lists.value, templates.value] = await Promise.all([api.lists(), api.templates()])
    suggestions.value = (await api.stats()).items.map((i) => i.name)
    if (!active.value && openLists.value.length) emit('select', openLists.value[0].id)
  } catch (e) {
    error.value = e.message
  }
}
onMounted(load)

function replace(arr, obj) {
  const i = arr.findIndex((x) => x.id === obj.id)
  if (i >= 0) arr[i] = obj
  else arr.unshift(obj)
}

async function newBlank() {
  const l = await api.createList('', [])
  replace(lists.value, l)
  emit('select', l.id)
}

async function newFrom(t) {
  const l = await api.newListFromTemplate(t.id)
  replace(lists.value, l)
  editingTemplate.value = null
  emit('select', l.id)
}

async function saveItems(items) {
  replace(lists.value, await api.updateList(active.value.id, { items }))
}

async function rename(name) {
  if (name.trim()) replace(lists.value, await api.updateList(active.value.id, { name }))
}

async function saveAsTemplate() {
  const name = prompt('Template name', active.value.name.replace(/ · .*$/, ''))
  if (!name) return
  replace(templates.value, await api.saveAsTemplate(active.value.id, name))
}

// For people not using Shop mode: everything on the list counts as bought.
async function markBought() {
  replace(lists.value, await api.completeList(active.value.id))
  const next = openLists.value[0]
  emit('select', next ? next.id : '')
}

async function removeList() {
  if (!confirm(`Delete "${active.value.name}"?`)) return
  await api.deleteList(active.value.id)
  lists.value = lists.value.filter((l) => l.id !== active.value.id)
  emit('select', openLists.value[0]?.id || '')
}

async function saveTemplateItems(items) {
  editingTemplate.value = await api.updateTemplate(editingTemplate.value.id, { items })
  replace(templates.value, editingTemplate.value)
}

async function removeTemplate(t) {
  if (!confirm(`Delete template "${t.name}"?`)) return
  await api.deleteTemplate(t.id)
  templates.value = templates.value.filter((x) => x.id !== t.id)
  editingTemplate.value = null
}
</script>

<template>
  <p v-if="error" class="error">{{ error }}</p>

  <!-- Editing a template -->
  <section v-if="editingTemplate">
    <h2>
      <span class="grow">📋 {{ editingTemplate.name }}</span>
      <button class="ghost" @click="editingTemplate = null">Done</button>
    </h2>
    <p class="muted small">Your base list. Each week starts as a copy of this.</p>
    <ItemsEditor :items="editingTemplate.items" :suggestions="suggestions" @change="saveTemplateItems" />
    <div class="row">
      <button class="primary" @click="newFrom(editingTemplate)">Start this week's list</button>
      <button class="ghost" @click="removeTemplate(editingTemplate)">Delete template</button>
    </div>
  </section>

  <template v-else>
    <section>
      <div class="store-picker">
        <select :value="activeListId" aria-label="Shopping list" @change="emit('select', $event.target.value)">
          <option disabled value="">{{ openLists.length ? 'Choose a list…' : 'No open lists' }}</option>
          <option v-for="l in openLists" :key="l.id" :value="l.id">{{ l.name }} · {{ l.items.length }} items</option>
        </select>
        <button class="icon" title="New blank list" @click="newBlank">＋</button>
      </div>

      <template v-if="active">
        <input class="title-input" :value="active.name" aria-label="List name" @change="rename($event.target.value)" />
        <p v-if="fromTemplate" class="muted small">From template “{{ fromTemplate.name }}” — tweak it for this week.</p>
        <ItemsEditor :items="active.items" :suggestions="suggestions" @change="saveItems" />
        <button class="primary block" :disabled="!active.items.length" @click="emit('shop')">🛒 Sort &amp; shop</button>
        <div class="row">
          <button class="ghost" @click="saveAsTemplate">📋 Save as template</button>
          <button class="ghost" @click="markBought">✓ Mark as bought</button>
          <button class="ghost" @click="removeList">Delete</button>
        </div>
      </template>
    </section>

    <section>
      <h2>Templates</h2>
      <p v-if="!templates.length" class="muted small">
        Save a list as a template to reuse it every week.
      </p>
      <ul class="list">
        <li v-for="t in templates" :key="t.id">
          <span class="grow">
            <strong>{{ t.name }}</strong>
            <span class="muted small"> · {{ t.items.length }} items</span>
          </span>
          <span class="actions wide">
            <button class="ghost" title="Edit template" @click="editingTemplate = t">✎</button>
            <button @click="newFrom(t)">New list</button>
          </span>
        </li>
      </ul>
    </section>
  </template>
</template>
