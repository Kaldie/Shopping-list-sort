<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  store: { type: Object, required: true },
  listId: { type: String, default: '' },
})
const emit = defineEmits(['updated', 'list-done'])

const raw = ref('')
const openLists = ref([])
const forList = ref(props.listId) // which list this receipt belongs to; '' = none
const lines = ref([]) // { line, item, matched, order, bought } in scan order
const busy = ref('')
const error = ref('')
const done = ref('')

const bought = computed(() => lines.value.filter((l) => l.bought))
const ordered = computed(() => bought.value.filter((l) => l.order))

onMounted(async () => {
  openLists.value = (await api.lists()).filter((l) => !l.completedAt)
  if (!openLists.value.some((l) => l.id === forList.value)) forList.value = openLists.value[0]?.id || ''
})

async function fromFile(e) {
  const f = e.target.files[0]
  if (!f) return
  busy.value = f.type === 'application/pdf' ? 'Reading PDF…' : 'Reading receipt…'
  error.value = ''
  try {
    raw.value = (await api.ocr(f)).join('\n')
    await parse()
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = ''
    e.target.value = ''
  }
}

async function parse() {
  done.value = ''
  error.value = ''
  const matches = await api.parseReceipt(raw.value, forList.value)
  lines.value = matches.map((m) => ({ ...m, order: true, bought: true }))
  if (!lines.value.length) error.value = 'No items found on this receipt.'
}

async function apply() {
  if (!bought.value.length) {
    error.value = 'Nothing marked as bought.'
    return
  }
  const res = await api.applyReceipt(
    props.store.id,
    forList.value,
    lines.value.map(({ item, order, bought }) => ({ item, order, bought })),
  )
  emit('updated', res.store)
  done.value = `Saved ${bought.value.length} bought items` +
    (ordered.value.length >= 2 ? ` and learned the route from ${ordered.value.length}.` : '.')
  lines.value = []
  raw.value = ''
  if (forList.value) emit('list-done')
}
</script>

<template>
  <section>
    <h2>Receipt</h2>
    <p class="muted">
      Self-scan receipts list items in the order you picked them. Upload the
      PDF from the app, or a photo — it counts as bought and teaches the route.
    </p>
    <label class="file">
      🧾 Upload receipt (PDF or photo)
      <input type="file" accept="application/pdf,image/*" @change="fromFile" />
    </label>
    <details class="paste">
      <summary class="muted small">…or paste the receipt text</summary>
      <textarea v-model="raw" rows="6" placeholder="AH BANANEN   1,99&#10;AH HALFV MELK 1L   1,19" />
      <button class="ghost" @click="parse">Read items</button>
    </details>
    <label class="field">
      <span class="small muted">Which list was this?</span>
      <select v-model="forList" @change="raw && parse()">
        <option v-for="l in openLists" :key="l.id" :value="l.id">{{ l.name }} · {{ l.items.length }} items</option>
        <option value="">None — just record what I bought</option>
      </select>
    </label>
    <p v-if="busy" class="muted">{{ busy }}</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="done" class="success">✓ {{ done }}</p>
  </section>

  <section v-if="lines.length">
    <h2>Scan order <small class="muted">{{ bought.length }}/{{ lines.length }} bought</small></h2>
    <p class="muted small">
      <span class="legend"><span class="dot-mini" /> route</span> tap if you went back for it ·
      <span class="legend">🛒 bought</span> tap if you didn't actually buy it.
    </p>
    <ol class="list numbered">
      <li v-for="(l, i) in lines" :key="i" :class="{ excluded: !l.bought }">
        <button
          class="dot"
          :class="{ off: !l.order || !l.bought }"
          :disabled="!l.bought"
          :title="l.order ? 'Used for the route — tap if you went back for it' : 'Not used for the route'"
          :aria-pressed="!l.order"
          @click="l.order = !l.order"
        />
        <span class="grow">
          <input v-model="l.item" class="inline" :aria-label="`Item for ${l.line}`" />
          <div class="muted small line">
            {{ l.matched ? '↔' : '+' }} {{ l.line }}<template v-if="!l.matched && forList"> · not on list</template>
          </div>
        </span>
        <button
          class="toggle"
          :class="{ off: !l.bought }"
          :title="l.bought ? 'Bought — tap if you didn\'t buy this' : 'Not bought'"
          :aria-pressed="!l.bought"
          @click="l.bought = !l.bought"
        >
          {{ l.bought ? '🛒' : '✕' }}
        </button>
      </li>
    </ol>
    <button class="primary block" @click="apply">Save receipt</button>
  </section>
</template>
