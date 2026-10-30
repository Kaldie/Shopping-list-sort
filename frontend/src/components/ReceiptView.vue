<script setup>
import { ref, computed, watch } from 'vue'
import { api } from '../api.js'

const props = defineProps({ store: { type: Object, required: true } })
const emit = defineEmits(['updated'])

const raw = ref('')
const items = ref([]) // { name, use } in scan order
const busy = ref('')
const error = ref('')
const done = ref('')

watch(() => props.store.id, reset)

const used = computed(() => items.value.filter((i) => i.use).map((i) => i.name))

function reset() {
  raw.value = ''
  items.value = []
  error.value = ''
}

async function fromPhoto(e) {
  const f = e.target.files[0]
  if (!f) return
  busy.value = 'Reading receipt…'
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
  items.value = (await api.parseReceipt(raw.value)).map((name) => ({ name, use: true }))
}

async function record() {
  if (used.value.length < 2) {
    error.value = 'Keep at least two items to learn an order.'
    return
  }
  const s = await api.trip(props.store.id, used.value)
  emit('updated', s)
  done.value = `Learned from ${used.value.length} items.`
  reset()
}
</script>

<template>
  <section>
    <h2>Self-scan receipt</h2>
    <p class="muted">
      With a hand scanner your receipt lists items in the order you picked
      them — a free walk through the store.
    </p>
    <label class="file">
      📷 Upload receipt photo
      <input type="file" accept="image/*" capture="environment" @change="fromPhoto" />
    </label>
    <p class="muted small">…or paste the receipt text:</p>
    <textarea v-model="raw" rows="6" placeholder="BANANEN   1,99&#10;HALFVOLLE MELK   1,19" />
    <button @click="parse">Read items</button>
    <p v-if="busy" class="muted">{{ busy }}</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="done" class="muted">✓ {{ done }}</p>
  </section>

  <section v-if="items.length">
    <h2>Scan order <small class="muted">{{ used.length }}/{{ items.length }} used</small></h2>
    <p class="muted small">
      Went back for something you forgot? Tap its dot so it isn't used for ordering.
    </p>
    <ol class="list">
      <li v-for="(it, i) in items" :key="i" :class="{ excluded: !it.use }">
        <button
          class="dot"
          :class="{ off: !it.use }"
          :title="it.use ? 'Used for ordering — tap to exclude' : 'Not used for ordering — tap to include'"
          :aria-pressed="!it.use"
          @click="it.use = !it.use"
        />
        <input v-model="it.name" class="inline" />
      </li>
    </ol>
    <button @click="record">Learn from this receipt</button>
  </section>
</template>
