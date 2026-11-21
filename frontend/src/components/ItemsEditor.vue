<script setup>
import { ref, nextTick, onMounted, onBeforeUnmount } from 'vue'
import Sortable from 'sortablejs'
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
const fresh = ref(new Set()) // names added or raised by the last add, highlighted
const seen = ref(new Set()) // pasted names that were already on the list
let timer

const update = (items) => emit('change', mergeItems(items))

// Adds lines and reports what happened: new items vs. counters raised.
// With keepExisting (paste), items already on the list are left alone, so
// pasting the same list twice changes nothing; repeats inside one paste still
// merge into a counter.
function addLines(lines, keepExisting = false) {
  let parsed = mergeItems(lines.map(parseItem))
  if (!parsed.length) return
  const have = new Set(props.items.map((i) => i.name.toLowerCase()))
  const names = [...new Set(parsed.map((p) => p.name.toLowerCase()))]
  const added = names.filter((n) => !have.has(n))
  const bumped = names.length - added.length
  if (keepExisting) parsed = parsed.filter((p) => !have.has(p.name.toLowerCase()))
  notice.value = { added: added.length, bumped, before: props.items.map((i) => ({ ...i })) }
  // Clear first so a repeat within the highlight window flashes again.
  fresh.value = new Set()
  seen.value = new Set()
  const flashNew = new Set(keepExisting ? added : names)
  const flashSeen = new Set(keepExisting ? names.filter((n) => have.has(n)) : [])
  nextTick(() => {
    fresh.value = flashNew
    seen.value = flashSeen
  })
  clearTimeout(timer)
  timer = setTimeout(() => {
    notice.value = null
    fresh.value = new Set()
    seen.value = new Set()
  }, 8000)
  update([...props.items, ...parsed])
}

function undo() {
  emit('change', notice.value.before)
  notice.value = null
  fresh.value = new Set()
  seen.value = new Set()
}

function add() {
  if (!toLines(newName.value).length) return
  addLines(toLines(newName.value))
  newName.value = ''
}

// Pasting several lines adds them all at once; one line is typed as usual.
function onPaste(e) {
  const text = e.clipboardData?.getData('text') || ''
  if (toLines(text).length < 2) return
  e.preventDefault()
  addLines(toLines(text), true)
}

// Fallback for inputs that bypass the paste event (drop, some mobile keyboards,
// autofill): the browser flattens newlines in a single-line input, but some
// keep them in the value, so split whatever arrives.
function onInput(e) {
  const v = e.target.value
  if (toLines(v).length < 2) return
  newName.value = ''
  addLines(toLines(v), true)
}

// Drag to reorder by the grip (touch works too). SortableJS moves the DOM
// node itself, so put it back and let Vue re-render from the new array.
const listEl = ref(null)
let sortable
onMounted(() => {
  sortable = Sortable.create(listEl.value, {
    handle: '.handle',
    draggable: 'li[data-item]',
    animation: 150,
    onEnd({ oldIndex, newIndex, item, from }) {
      if (oldIndex === newIndex) return
      from.removeChild(item)
      from.insertBefore(item, from.children[oldIndex] || null)
      const next = [...props.items]
      next.splice(newIndex, 0, next.splice(oldIndex, 1)[0])
      emit('change', next)
    },
  })
})
onBeforeUnmount(() => sortable?.destroy())

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
  <ul ref="listEl" class="list items">
    <li v-for="(it, i) in items" :key="it.name" data-item :class="{ fresh: fresh.has(it.name.toLowerCase()), seen: seen.has(it.name.toLowerCase()) }">
      <span class="handle" :title="t('reorder')" aria-hidden="true">⠿</span>
      <input class="inline grow" :value="it.name" @change="rename(i, $event.target.value)" />
      <span class="stepper">
        <button class="ghost" :aria-label="t('less', it.name)" @click="setQty(i, it.qty - 1)">−</button>
        <input
          class="qty-input"
          type="number"
          min="0"
          inputmode="numeric"
          :aria-label="t('qty')"
          :value="it.qty"
          @change="setQty(i, Math.floor(Number($event.target.value)) || 0)"
        />
        <button class="ghost" :aria-label="t('more', it.name)" @click="setQty(i, it.qty + 1)">+</button>
      </span>
    </li>
    <li v-if="!items.length" class="muted">{{ t('noItems') }}</li>
  </ul>
  <form class="row" @submit.prevent="add">
    <textarea
      v-model="newName"
      class="add-input"
      rows="1"
      list="item-suggestions"
      :placeholder="t('addItem')"
      autocomplete="off"
      enterkeyhint="done"
      @keydown.enter.prevent="add"
      @paste="onPaste"
      @input="onInput"
    />
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
