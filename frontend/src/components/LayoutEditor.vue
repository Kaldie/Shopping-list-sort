<script setup>
import { ref, watch, nextTick, onBeforeUnmount } from 'vue'
import Sortable from 'sortablejs'
import { api } from '../api.js'
import { t } from '../i18n.js'

const props = defineProps({
  store: { type: Object, required: true },
  userId: { type: String, required: true },
})
const emit = defineEmits(['updated'])

const sections = ref([])
const newSection = ref('')
const layout = ref(null)
const busy = ref('')
const error = ref('')
const board = ref(null) // section cards, reorderable
const unplacedEl = ref(null) // items without a section, drag-only
let sortables = []

async function refresh() {
  sections.value = [...(props.store.sections || [])]
  layout.value = await api.layout(props.store.id)
}
watch(() => [props.store.id, props.store.forked], refresh, { immediate: true })

async function save() {
  const s = await api.updateStore(props.store.id, { sections: sections.value })
  emit('updated', s)
  layout.value = await api.layout(props.store.id)
}

// Drag and drop (SortableJS, works with touch: long-press an item). After a
// drop the DOM is put back and the data refreshed, so Vue stays in charge.
function destroyDnD() {
  sortables.forEach((x) => x.destroy())
  sortables = []
}

function initDnD() {
  destroyDnD()
  if (board.value) {
    sortables.push(
      Sortable.create(board.value, { draggable: '.sec', handle: '.handle', animation: 150, onEnd: onSectionDrop }),
    )
  }
  const zones = [...(board.value?.querySelectorAll('.chips') || []), ...(unplacedEl.value ? [unplacedEl.value] : [])]
  for (const el of zones) {
    sortables.push(
      Sortable.create(el, {
        group: { name: 'items', put: !el.classList.contains('source') },
        draggable: '.chip',
        sort: false,
        animation: 150,
        delay: 150,
        delayOnTouchOnly: true,
        onEnd: onItemDrop,
      }),
    )
  }
}
watch(layout, async () => {
  await nextTick()
  initDnD()
})
onBeforeUnmount(destroyDnD)

async function onSectionDrop(evt) {
  const names = [...board.value.children].filter((e) => e.classList.contains('sec')).map((e) => e.dataset.name)
  const from = evt.from
  from.insertBefore(evt.item, evt.oldIndex < evt.newIndex ? from.children[evt.oldIndex] : from.children[evt.oldIndex + 1] ?? null)
  if (names.join('|') === layout.value.sections.map((x) => x.name).join('|')) return
  sections.value = names
  try {
    await save()
  } catch (e) {
    error.value = e.message
  }
}

async function onItemDrop(evt) {
  if (evt.to === evt.from) return
  const item = evt.item.dataset.item
  const section = evt.to.dataset.section
  evt.from.insertBefore(evt.item, evt.from.children[evt.oldIndex] ?? null)
  error.value = ''
  try {
    emit('updated', await api.updateStore(props.store.id, { itemSection: { [item]: section } }))
    layout.value = await api.layout(props.store.id)
  } catch (e) {
    error.value = e.message
  }
}

function move(i, d) {
  const j = i + d
  if (j < 0 || j >= sections.value.length) return
  const arr = sections.value
  ;[arr[i], arr[j]] = [arr[j], arr[i]]
  save()
}

function removeAt(i) {
  sections.value.splice(i, 1)
  save()
}

function add() {
  const name = newSection.value.trim()
  if (!name) return
  sections.value.push(name)
  newSection.value = ''
  save()
}

// Aisle-sign photos, selected in walking order: each photo's first OCR line
// becomes the next section.
async function importPhotos(e) {
  const files = [...e.target.files]
  error.value = ''
  for (const [i, f] of files.entries()) {
    busy.value = t('readingPhoto', i + 1, files.length)
    try {
      const lines = await api.ocr(f)
      if (lines.length) sections.value.push(lines[0])
    } catch (err) {
      error.value = err.message
      break
    }
  }
  busy.value = ''
  e.target.value = ''
  save()
}

async function unfork() {
  if (!confirm(t('unforkConfirm'))) return
  emit('updated', await api.unforkStore(props.store.id))
  await refresh()
}

function adoptLearned() {
  sections.value = layout.value.sections.map((s) => s.name)
  save()
}
</script>

<template>
  <div class="banner" :class="{ own: store.forked }">
    <template v-if="store.forked">
      <span class="grow">{{ t('ownRoute') }}</span>
      <button class="ghost" @click="unfork">{{ t('useShared') }}</button>
    </template>
    <span v-else>{{ t('sharedLayout') }}</span>
  </div>

  <section>
    <h2>{{ t('layoutNow') }}</h2>
    <p class="muted small">{{ t('layoutNowHint', store.trips) }}</p>
    <p v-if="layout && !layout.sections.length" class="muted">{{ t('layoutEmpty') }}</p>
    <p v-if="layout && layout.sections.length" class="muted small">{{ t('dragHint') }}</p>
    <div v-if="layout" ref="board" class="board">
      <div v-for="(s, i) in layout.sections" :key="s.name" class="sec" :data-name="s.name">
        <div class="sec-head">
          <span class="handle" :title="t('dragSection')">⠿</span>
          <strong class="grow">{{ i + 1 }}. {{ s.name }}</strong>
          <span class="badge" :class="s.source">{{ t('source' + s.source[0].toUpperCase() + s.source.slice(1)) }}</span>
        </div>
        <div class="chips" :data-section="s.name">
          <span v-for="it in s.items" :key="it" class="chip" :data-item="it">{{ it }}</span>
        </div>
      </div>
    </div>
    <div v-if="layout && layout.unplaced.length" class="sec unplaced">
      <div class="sec-head">
        <strong class="grow">{{ t('unplaced') }}</strong>
      </div>
      <p class="muted small">{{ t('unplacedHint') }}</p>
      <div ref="unplacedEl" class="chips source" data-section="">
        <span v-for="it in layout.unplaced" :key="it" class="chip" :data-item="it">{{ it }}</span>
      </div>
    </div>
    <details v-if="layout && layout.items.length" class="paste">
      <summary class="muted small">{{ t('itemOrder', layout.items.length) }}</summary>
      <ol class="list numbered">
        <li v-for="it in layout.items" :key="it">{{ it }}</li>
      </ol>
    </details>
  </section>

  <details class="paste">
    <summary>{{ t('editSections') }}</summary>
    <p class="muted small">{{ t('editHint') }}</p>
    <ol class="list numbered">
      <li v-for="(s, i) in sections" :key="s + i">
        <span class="grow">{{ s }}</span>
        <span class="actions">
          <button class="ghost" @click="move(i, -1)">↑</button>
          <button class="ghost" @click="move(i, 1)">↓</button>
          <button class="ghost" @click="removeAt(i)">✕</button>
        </span>
      </li>
    </ol>
    <form class="row" @submit.prevent="add">
      <input v-model="newSection" :placeholder="t('sectionPlaceholder')" />
      <button>{{ t('add') }}</button>
    </form>
    <button v-if="layout && layout.sections.length" class="ghost" @click="adoptLearned">{{ t('adopt') }}</button>
    <label class="file">
      {{ t('importPhotos') }}
      <input type="file" accept="image/*" multiple @change="importPhotos" />
    </label>
    <p v-if="busy" class="muted">{{ busy }}</p>
    <p v-if="error" class="error">{{ error }}</p>
  </details>
</template>
