<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n.js'

const emit = defineEmits(['signed-in'])
const config = ref(null)
const devName = ref('')
const error = ref('')
const googleButton = ref(null)

async function finish(promise) {
  error.value = ''
  try {
    emit('signed-in', await promise)
  } catch (e) {
    error.value = e.message
  }
}

// Google Identity Services: renders the official button and hands us an ID
// token, which the backend verifies.
function loadGoogle(clientId) {
  const script = document.createElement('script')
  script.src = 'https://accounts.google.com/gsi/client'
  script.async = true
  script.onload = () => {
    window.google.accounts.id.initialize({
      client_id: clientId,
      callback: (res) => finish(api.loginGoogle(res.credential)),
    })
    window.google.accounts.id.renderButton(googleButton.value, {
      theme: 'outline', size: 'large', shape: 'pill', text: 'continue_with',
    })
  }
  script.onerror = () => (error.value = t('googleFail'))
  document.head.appendChild(script)
}

onMounted(async () => {
  try {
    config.value = await api.authConfig()
    if (config.value.googleClientId) loadGoogle(config.value.googleClientId)
  } catch (e) {
    error.value = e.message
  }
})
</script>

<template>
  <div class="signin">
    <h1>{{ t('appName') }}</h1>
    <p class="muted">{{ t('tagline') }}</p>

    <div v-if="config?.googleClientId" ref="googleButton" class="google-btn" />

    <form v-else-if="config?.devLogin" class="row" @submit.prevent="finish(api.loginDev(devName))">
      <input v-model="devName" :placeholder="t('signInDev')" />
      <button>{{ t('signIn') }}</button>
    </form>
    <p v-if="error" class="error">{{ error }}</p>
  </div>
</template>
