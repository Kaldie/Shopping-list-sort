<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n.js'

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
  busy.value = f.type === 'application/pdf' ? t('readingPdf') : t('readingReceipt')
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
  let matches
  try {
    matches = await api.parseReceipt(raw.value, forList.value)
  } catch (err) {
    error.value = err.message === 'not found' ? t('outOfDate') : err.message
    return
  }
  lines.value = matches.map((m) => ({ ...m, order: true, bought: true }))
  if (!lines.value.length) error.value = t('noItemsOnReceipt')
}

async function apply() {
  if (!bought.value.length) {
    error.value = t('nothingBought')
    return
  }
  error.value = ''
  let res
  try {
    res = await api.applyReceipt(
      props.store.id,
      forList.value,
      lines.value.map(({ item, order, bought }) => ({ item, order, bought })),
    )
  } catch (err) {
    // 404: this page is out of date (store or list was deleted elsewhere).
    error.value = err.message === 'not found' ? t('outOfDate') : err.message
    return
  }
  emit('updated', res.store)
  done.value = t('receiptSaved', bought.value.length, ordered.value.length)
  lines.value = []
  raw.value = ''
  if (forList.value) emit('list-done')
}
</script>

<template>
  <section>
    <h2>{{ t('receipt') }}</h2>
    <label class="file">
      {{ t('uploadReceipt') }}
      <input type="file" accept="application/pdf,image/*" @change="fromFile" />
    </label>
    <details class="paste">
      <summary class="muted small">{{ t('pasteReceipt') }}</summary>
      <textarea v-model="raw" rows="6" placeholder="AH BANANEN   1,99&#10;AH HALFV MELK 1L   1,19" />
      <button class="ghost" @click="parse">{{ t('readItems') }}</button>
    </details>
    <label class="field">
      <span class="small muted">{{ t('whichList') }}</span>
      <select v-model="forList" @change="raw && parse()">
        <option v-for="l in openLists" :key="l.id" :value="l.id">{{ l.name }} · {{ t('items', l.items.length) }}</option>
        <option value="">{{ t('noList') }}</option>
      </select>
    </label>
    <p v-if="busy" class="muted">{{ busy }}</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="done" class="success">{{ done }}</p>
  </section>

  <section v-if="lines.length">
    <h2>{{ t('scanOrder') }} <small class="muted">{{ t('boughtCount', bought.length, lines.length) }}</small></h2>
    <p class="muted small">{{ t('routeHint') }}</p>
    <ol class="list numbered">
      <li v-for="(l, i) in lines" :key="i" :class="{ excluded: !l.bought }">
        <button
          class="dot"
          :class="{ off: !l.order || !l.bought }"
          :disabled="!l.bought"
          :aria-pressed="!l.order"
          @click="l.order = !l.order"
        />
        <span class="grow">
          <input v-model="l.item" class="inline" :aria-label="l.line" />
          <div class="muted small line">
            {{ l.line }}<template v-if="!l.matched && forList"> · {{ t('notOnList') }}</template>
          </div>
        </span>
        <button class="toggle" :class="{ off: !l.bought }" :aria-pressed="!l.bought" @click="l.bought = !l.bought">
          {{ l.bought ? '✓' : '✕' }}
        </button>
      </li>
    </ol>
    <button class="primary block" @click="apply">{{ t('saveReceipt') }}</button>
  </section>
</template>
