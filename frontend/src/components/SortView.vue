<script setup>
import { ref, computed, watch, onBeforeUnmount, nextTick } from 'vue'
import Sortable from 'sortablejs'
import { api } from '../api.js'
import { t } from '../i18n.js'

const props = defineProps({
  store: { type: Object, required: true },
  listId: { type: String, default: '' },
})
const emit = defineEmits(['updated', 'edit-list', 'list-done'])

const list = ref(null)
const groups = ref([])
const picked = ref([]) // item names in the order they were checked off
const error = ref('')

const qty = computed(() => Object.fromEntries((list.value?.items || []).map((i) => [i.name, i.qty])))
// Section headings are optional; the default is one plain list in walking order.
const showSections = ref(localStorage.getItem('showSections') === '1')
watch(showSections, (v) => localStorage.setItem('showSections', v ? '1' : '0'))

// Ticked items stay visible (struck through) for a short while, so a mis-tap
// can be undone, then drop off the list. "Show ticked" brings them back.
const HIDE_AFTER_MS = 30_000
const hidden = ref({}) // item name → true once its grace period is over
const showTicked = ref(false)
const timers = {}

function clearTimers() {
  for (const k of Object.keys(timers)) {
    clearTimeout(timers[k])
    delete timers[k]
  }
}
onBeforeUnmount(clearTimers)

const hiddenCount = computed(() => Object.keys(hidden.value).length)
const visibleGroups = computed(() =>
  groups.value
    .map((g) => ({ ...g, items: g.items.filter((i) => showTicked.value || !hidden.value[i]) }))
    .filter((g) => g.items.length),
)

const flat = computed(() => groups.value.flatMap((g) => g.items))
const total = computed(() => flat.value.length)

// Drag by the grip to change the order. The new order is saved as this
// supermarket's item order (like dragging on the Layout page), so the next
// list is sorted the same way.
const routeEl = ref(null)
let sortables = []
function destroyDnD() {
  sortables.forEach((x) => x.destroy())
  sortables = []
}
function initDnD() {
  destroyDnD()
  if (!routeEl.value) return
  for (const el of routeEl.value.querySelectorAll('.group-items')) {
    sortables.push(
      Sortable.create(el, {
        group: 'route',
        handle: '.grip',
        draggable: '.check',
        animation: 150,
        // Pointer-based dragging: same behaviour for touch and mouse, and the
        // item can be dragged across section headings.
        forceFallback: true,
        fallbackOnBody: true,
        onEnd: onDrop,
      }),
    )
  }
}
watch(visibleGroups, () => nextTick(initDnD), { flush: 'post' })
onBeforeUnmount(destroyDnD)

// Places the visible list's new order into the store's full item order: list
// items take the slots they already had, other items stay put, and items the
// store didn't know yet go right after the item dragged before them.
function mergeOrder(full, visible) {
  const vis = visible.map((n) => n.trim().toLowerCase())
  const inFull = new Set(full)
  const known = vis.filter((n) => inFull.has(n))
  const set = new Set(known)
  let k = 0
  const out = full.map((n) => (set.has(n) ? known[k++] : n))
  vis.forEach((n, i) => {
    if (inFull.has(n)) return
    const at = i ? out.indexOf(vis[i - 1]) + 1 : Math.max(0, out.indexOf(vis[i + 1]))
    out.splice(at, 0, n)
  })
  return out
}

async function onDrop(evt) {
  const names = [...routeEl.value.querySelectorAll('.check[data-item]')].map((e) => e.dataset.item)
  // SortableJS moved the DOM node; put it back and let Vue re-render.
  evt.item.remove()
  evt.from.insertBefore(evt.item, evt.from.children[evt.oldIndex] || null)
  if (evt.from === evt.to && evt.oldIndex === evt.newIndex) return
  error.value = ''
  try {
    const layout = await api.layout(props.store.id)
    emit('updated', await api.updateStore(props.store.id, { itemOrder: mergeOrder(layout.items || [], names) }))
    groups.value = (await api.sort(props.store.id, list.value.items.map((i) => i.name))).groups
  } catch (e) {
    error.value = e.message
  }
}

async function load() {
  error.value = ''
  groups.value = []
  picked.value = []
  hidden.value = {}
  clearTimers()
  list.value = null
  if (!props.listId) return
  try {
    list.value = await api.getList(props.listId)
    groups.value = (await api.sort(props.store.id, list.value.items.map((i) => i.name))).groups
  } catch (e) {
    error.value = e.message
  }
}
watch(() => [props.store.id, props.listId], load, { immediate: true })

function toggle(item) {
  const i = picked.value.indexOf(item)
  if (i >= 0) {
    picked.value.splice(i, 1)
    clearTimeout(timers[item])
    delete timers[item]
    delete hidden.value[item]
  } else {
    picked.value.push(item)
    timers[item] = setTimeout(() => {
      hidden.value[item] = true
      delete timers[item]
    }, HIDE_AFTER_MS)
  }
}

// The check-off order is an observed walk through the store: record it as a
// trip, and complete the list with what was actually bought (for statistics).
// An item with a count is struck once, whatever the quantity.
async function finish() {
  if (!picked.value.length) {
    error.value = t('tickFirst')
    return
  }
  if (picked.value.length < total.value && !confirm(t('notAllTicked'))) return
  if (picked.value.length >= 2) emit('updated', await api.trip(props.store.id, picked.value))
  await api.completeList(list.value.id, picked.value.map((name) => ({ name, qty: qty.value[name] || 1 })))
  emit('list-done')
}
</script>

<template>
  <section v-if="!listId || !list">
    <p v-if="error" class="error">{{ error }}</p>
    <div class="empty">
      <p>{{ t('noListSelected') }}</p>
      <button @click="emit('edit-list')">{{ t('makeList') }}</button>
    </div>
  </section>

  <section v-else>
    <h2>
      <span class="grow">{{ list.name }} <small class="muted">{{ picked.length }}/{{ total }}</small></span>
      <button class="ghost" @click="emit('edit-list')">{{ t('edit') }}</button>
    </h2>
    <div class="progress"><div :style="{ width: total ? (100 * picked.length) / total + '%' : 0 }" /></div>
    <p v-if="error" class="error">{{ error }}</p>
    <label class="muted small">
      <input v-model="showSections" type="checkbox" /> {{ t('showSections') }}
    </label>
    <button v-if="hiddenCount" class="ghost show-ticked" :aria-pressed="showTicked" @click="showTicked = !showTicked">
      {{ showTicked ? t('hideTicked') : t('showTicked', hiddenCount) }}
    </button>
    <p v-if="total && !visibleGroups.length" class="muted all-done">{{ t('allInCart') }}</p>
    <div ref="routeEl">
    <div v-for="g in visibleGroups" :key="g.section" class="group">
      <h3 v-if="showSections">{{ g.section }}</h3>
      <TransitionGroup name="tick" tag="div" class="group-items">
        <label v-for="item in g.items" :key="item" class="check" :data-item="item" :class="{ done: picked.includes(item) }">
          <span class="grip" :title="t('reorder')" aria-hidden="true" @click.prevent>⠿</span>
          <input type="checkbox" :checked="picked.includes(item)" @change="toggle(item)" />
          <span class="grow">{{ item }}</span>
          <span v-if="qty[item] > 1" class="count">{{ qty[item] }}×</span>
        </label>
      </TransitionGroup>
    </div>
    </div>
    <button class="primary block sticky-action" @click="finish">{{ t('finish') }}</button>
  </section>
</template>
