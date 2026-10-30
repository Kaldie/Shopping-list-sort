<script setup>
import { ref } from 'vue'
import { api } from '../api.js'

// Editable list of { name, qty }. Emits 'change' with the new array.
const props = defineProps({
  items: { type: Array, required: true },
  suggestions: { type: Array, default: () => [] }, // item names for autocomplete
})
const emit = defineEmits(['change'])

const newName = ref('')
const newQty = ref(1)
const busy = ref('')
const error = ref('')

const update = (items) => emit('change', items)

function add() {
  const name = newName.value.trim()
  if (!name) return
  update([...props.items, { name, qty: newQty.value || 1 }])
  newName.value = ''
  newQty.value = 1
}

function setQty(i, qty) {
  if (qty < 1) return remove(i)
  update(props.items.map((it, j) => (j === i ? { ...it, qty } : it)))
}

function rename(i, name) {
  update(props.items.map((it, j) => (j === i ? { ...it, name } : it)))
}

function remove(i) {
  update(props.items.filter((_, j) => j !== i))
}

// Handwritten / printed list photo → one item per recognised line.
async function scan(e) {
  const f = e.target.files[0]
  if (!f) return
  busy.value = 'Reading list…'
  error.value = ''
  try {
    const lines = await api.ocr(f)
    update([...props.items, ...lines.map((name) => ({ name, qty: 1 }))])
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
    <li v-for="(it, i) in items" :key="i">
      <input class="inline grow" :value="it.name" @change="rename(i, $event.target.value)" />
      <span class="stepper">
        <button class="ghost" :aria-label="`Less ${it.name}`" @click="setQty(i, it.qty - 1)">−</button>
        <span class="qty">{{ it.qty }}</span>
        <button class="ghost" :aria-label="`More ${it.name}`" @click="setQty(i, it.qty + 1)">+</button>
      </span>
    </li>
    <li v-if="!items.length" class="muted">No items yet.</li>
  </ul>
  <form class="row" @submit.prevent="add">
    <input v-model="newName" list="item-suggestions" placeholder="Add item…" autocomplete="off" />
    <input v-model.number="newQty" type="number" min="1" class="qty-input" aria-label="Quantity" />
    <button>Add</button>
  </form>
  <datalist id="item-suggestions">
    <option v-for="s in suggestions" :key="s" :value="s" />
  </datalist>
  <label class="file">
    📷 Scan handwritten list
    <input type="file" accept="image/*" capture="environment" @change="scan" />
  </label>
  <p v-if="busy" class="muted">{{ busy }}</p>
  <p v-if="error" class="error">{{ error }}</p>
</template>
