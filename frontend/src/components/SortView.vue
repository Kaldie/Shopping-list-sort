<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue'
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
    <div v-for="g in visibleGroups" :key="g.section" class="group">
      <h3 v-if="showSections">{{ g.section }}</h3>
      <TransitionGroup name="tick">
        <label v-for="item in g.items" :key="item" class="check" :class="{ done: picked.includes(item) }">
          <input type="checkbox" :checked="picked.includes(item)" @change="toggle(item)" />
          <span class="grow">{{ item }}</span>
          <span v-if="qty[item] > 1" class="count">{{ qty[item] }}×</span>
        </label>
      </TransitionGroup>
    </div>
    <button class="primary block sticky-action" @click="finish">{{ t('finish') }}</button>
  </section>
</template>
