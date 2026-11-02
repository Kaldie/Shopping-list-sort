<script setup>
import { ref } from 'vue'
import { api, toLines, parseItem, mergeItems } from '../api.js'
import { t } from '../i18n.js'

// Editable list of { name, qty }. Emits 'change' with the new array.
// Everything typed, pasted or scanned goes through parseItem: alphanumeric
// names only, "2 melk" becomes a count, repeats merge into one counter.
const props = defineProps({
  items: { type: Array, required: true },
  suggestions: { type: Array, default: () => [] }, // item names for autocomplete
})
const emit = defineEmits(['change'])

const newName = ref('')
const busy = ref('')
const error = ref('')
const pasted = ref('')
const pasteOpen = ref(false)

const update = (items) => emit('change', mergeItems(items))
const addLines = (lines) => update([...props.items, ...lines.map(parseItem)])

function add() {
  addLines([newName.value])
  newName.value = ''
}

function setQty(i, qty) {
  if (qty < 1) return remove(i)
  update(props.items.map((it, j) => (j === i ? { ...it, qty } : it)))
}

function rename(i, name) {
  const p = parseItem(name)
  if (!p) return remove(i)
  update(props.items.map((it, j) => (j === i ? { name: p.name, qty: it.qty * p.qty } : it)))
}

function remove(i) {
  update(props.items.filter((_, j) => j !== i))
}

function addPasted() {
  addLines(toLines(pasted.value))
  pasted.value = ''
  pasteOpen.value = false
}

// Photo of a list → recognised lines land in the paste box, so OCR mistakes
// can be fixed before they become items.
async function scan(e) {
  const f = e.target.files[0]
  if (!f) return
  busy.value = t('readingList')
  error.value = ''
  try {
    const lines = await api.ocr(f)
    if (!lines.length) throw new Error(t('nothingRecognised'))
    pasted.value = [...toLines(pasted.value), ...lines].join('\n')
    pasteOpen.value = true
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = ''
    e.target.value = ''
  }
}
</script>

<template>
  <ul class="list items">
    <li v-for="(it, i) in items" :key="it.name">
      <input class="inline grow" :value="it.name" @change="rename(i, $event.target.value)" />
      <span class="stepper">
        <button class="ghost" :aria-label="t('less', it.name)" @click="setQty(i, it.qty - 1)">−</button>
        <span class="qty">{{ it.qty }}</span>
        <button class="ghost" :aria-label="t('more', it.name)" @click="setQty(i, it.qty + 1)">+</button>
      </span>
    </li>
    <li v-if="!items.length" class="muted">{{ t('noItems') }}</li>
  </ul>
  <form class="row" @submit.prevent="add">
    <input v-model="newName" list="item-suggestions" :placeholder="t('addItem')" autocomplete="off" />
    <button>{{ t('add') }}</button>
  </form>
  <datalist id="item-suggestions">
    <option v-for="s in suggestions" :key="s" :value="s" />
  </datalist>
  <label class="file">
    {{ t('scanList') }}
    <input type="file" accept="image/*" capture="environment" @change="scan" />
  </label>
  <details class="paste" :open="pasteOpen" @toggle="pasteOpen = $event.target.open">
    <summary class="muted small">{{ t('pasteList') }}</summary>
    <textarea v-model="pasted" rows="6" placeholder="Bananen&#10;2 Melk&#10;Brood" />
    <button class="ghost" @click="addPasted">{{ t('addItems') }}</button>
  </details>
  <p v-if="busy" class="muted">{{ busy }}</p>
  <p v-if="error" class="error">{{ error }}</p>
</template>
