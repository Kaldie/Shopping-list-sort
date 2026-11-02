<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n.js'
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

async function newFrom(tpl) {
  const l = await api.newListFromTemplate(tpl.id)
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
  const name = prompt(t('templateName'), active.value.name.replace(/ · .*$/, ''))
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
  if (!confirm(t('deleteListConfirm', active.value.name))) return
  await api.deleteList(active.value.id)
  lists.value = lists.value.filter((l) => l.id !== active.value.id)
  emit('select', openLists.value[0]?.id || '')
}

async function saveTemplateItems(items) {
  editingTemplate.value = await api.updateTemplate(editingTemplate.value.id, { items })
  replace(templates.value, editingTemplate.value)
}

async function removeTemplate(tpl) {
  if (!confirm(t('deleteTemplateConfirm', tpl.name))) return
  await api.deleteTemplate(tpl.id)
  templates.value = templates.value.filter((x) => x.id !== tpl.id)
  editingTemplate.value = null
}
</script>

<template>
  <p v-if="error" class="error">{{ error }}</p>

  <section v-if="editingTemplate">
    <h2>
      <span class="grow">{{ editingTemplate.name }}</span>
      <button class="ghost" @click="editingTemplate = null">{{ t('done') }}</button>
    </h2>
    <ItemsEditor :items="editingTemplate.items" :suggestions="suggestions" @change="saveTemplateItems" />
    <div class="row">
      <button class="primary" @click="newFrom(editingTemplate)">{{ t('startWeek') }}</button>
      <button class="ghost" @click="removeTemplate(editingTemplate)">{{ t('deleteTemplate') }}</button>
    </div>
  </section>

  <template v-else>
    <section>
      <div class="store-picker">
        <select :value="activeListId" aria-label="Lijst" @change="emit('select', $event.target.value)">
          <option disabled value="">{{ openLists.length ? t('chooseList') : t('noOpenLists') }}</option>
          <option v-for="l in openLists" :key="l.id" :value="l.id">{{ l.name }} · {{ t('items', l.items.length) }}</option>
        </select>
        <button class="icon" :title="t('newList')" @click="newBlank">+</button>
      </div>

      <template v-if="active">
        <input class="title-input" :value="active.name" :aria-label="t('listName')" @change="rename($event.target.value)" />
        <ItemsEditor :items="active.items" :suggestions="suggestions" @change="saveItems" />
        <button class="primary block" :disabled="!active.items.length" @click="emit('shop')">{{ t('shop') }}</button>
        <div class="row">
          <button class="ghost" @click="saveAsTemplate">{{ t('saveTemplate') }}</button>
          <button class="ghost" @click="markBought">{{ t('markBought') }}</button>
          <button class="ghost" @click="removeList">{{ t('delete') }}</button>
        </div>
      </template>
    </section>

    <section v-if="templates.length">
      <h2>{{ t('templates') }}</h2>
      <ul class="list">
        <li v-for="tpl in templates" :key="tpl.id">
          <span class="grow">
            <strong>{{ tpl.name }}</strong>
            <span class="muted small"> · {{ t('items', tpl.items.length) }}</span>
          </span>
          <span class="actions wide">
            <button class="ghost" @click="editingTemplate = tpl">{{ t('edit') }}</button>
            <button @click="newFrom(tpl)">{{ t('newFromTemplate') }}</button>
          </span>
        </li>
      </ul>
    </section>
  </template>
</template>
