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
const notice = ref(null) // { added, bumped, before } after an add, with undo
const fresh = ref(new Set()) // names added by the last add, highlighted
let timer

const update = (items) => emit('change', mergeItems(items))

// Adds lines and reports what happened: new items vs. counters raised.
function addLines(lines) {
  const parsed = lines.map(parseItem).filter(Boolean)
  if (!parsed.length) return
  const have = new Set(props.items.map((i) => i.name.toLowerCase()))
  const names = [...new Set(parsed.map((p) => p.name.toLowerCase()))]
  const added = names.filter((n) => !have.has(n))
  const bumped = names.length - added.length
  notice.value = { added: added.length, bumped, before: props.items.map((i) => ({ ...i })) }
  fresh.value = new Set(added)
  clearTimeout(timer)
  timer = setTimeout(() => {
    notice.value = null
    fresh.value = new Set()
  }, 8000)
  update([...props.items, ...parsed])
}

function undo() {
  emit('change', notice.value.before)
  notice.value = null
  fresh.value = new Set()
}

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
    <li v-for="(it, i) in items" :key="it.name" :class="{ fresh: fresh.has(it.name.toLowerCase()) }">
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
    <input v-model="newName" list="item-suggestions" :placeholder="t('addItem')" autocomplete="off" @paste="onPaste" />
    <button>{{ t('add') }}</button>
  </form>
  <datalist id="item-suggestions">
    <option v-for="s in suggestions" :key="s" :value="s" />
  </datalist>
  <p v-if="notice" class="notice" role="status">
    <span class="grow">{{ t('added', notice.added, notice.bumped) }}</span>
    <button class="ghost" @click="undo">{{ t('undo') }}</button>
  </p>
</template>
