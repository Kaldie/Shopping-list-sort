<script setup>
import { ref } from 'vue'
import { toLines, parseItem, mergeItems } from '../api.js'
import { t } from '../i18n.js'

// Editable list of { name, qty }. Emits 'change' with the new array.
// Everything typed or pasted goes through parseItem: alphanumeric
// names only, "2 melk" becomes a count, repeats merge into one counter.
const props = defineProps({
  items: { type: Array, required: true },
  suggestions: { type: Array, default: () => [] }, // item names for autocomplete
})
const emit = defineEmits(['change'])

const newName = ref('')

const update = (items) => emit('change', mergeItems(items))
const addLines = (lines) => update([...props.items, ...lines.map(parseItem)])

function add() {
  addLines(toLines(newName.value))
  newName.value = ''
}

// Pasting several lines adds them all at once; one line is typed as usual.
function onPaste(e) {
  const text = e.clipboardData?.getData('text') || ''
  if (toLines(text).length < 2) return
  e.preventDefault()
  addLines(toLines(text))
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
    <input v-model="newName" list="item-suggestions" @paste="onPaste" :placeholder="t('addItem')" autocomplete="off" />
    <button>{{ t('add') }}</button>
  </form>
  <datalist id="item-suggestions">
    <option v-for="s in suggestions" :key="s" :value="s" />
  </datalist>
</template>
