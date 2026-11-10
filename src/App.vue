<template>
  <div class="memorial-banner">
    <span class="memorial-icon">⚫</span>
    <span class="memorial-text">
      In loving memory of <strong>Mike Cornell</strong> (mcornell.dev) — 2026
    </span>
  </div>
  <AppHeader
    :user="currentUser"
    :loading="authLoading"
  />
  <RouterView />
  <footer class="site-footer">
    <div class="footer-content">
      <span class="footer-memorial">
        <span class="memorial-icon-small">⚫</span>
        Built with ❤️ by <a href="https://bsky.app/profile/mcornell.dev" target="_blank" rel="noopener">Mike Cornell</a>
      </span>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { ref, provide, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppHeader from './components/AppHeader.vue'
import { getGoogleRedirectResult } from './firebase'
import { flushGuestPredictions } from './guestPredictions'

const route = useRoute()
const router = useRouter()
const currentUser = ref<{ userID: string; handle: string; emailVerified: boolean } | null>(null)
const authLoading = ref(true)

provide('currentUser', currentUser)

async function fetchUser() {
  const res = await fetch('/api/me')
  currentUser.value = res.ok ? await res.json() : null
  authLoading.value = false
}

onMounted(async () => {
  try {
    const token = await getGoogleRedirectResult()
    if (token) {
      const res = await fetch('/auth/session', {
        method: 'POST',
        body: new URLSearchParams({ idToken: token }),
      })
      if (res.ok) {
        await flushGuestPredictions()
        await fetchUser()
        router.push('/matches')
        return
      }
    }
  } catch (err) {
    console.error('Google redirect result failed:', err)
  }
  await fetchUser()
})

watch(() => route.path, fetchUser)
</script>

<style scoped>
.memorial-banner {
  background: #0a0a0a;
  border-bottom: 1px solid rgba(254, 221, 0, 0.3);
  padding: 0.5rem 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
}

.memorial-icon {
  font-size: 1.2rem;
  opacity: 0.8;
}

.memorial-text {
  font-family: 'DM Mono', monospace;
  font-size: 0.8rem;
  color: #aaa;
  letter-spacing: 0.03em;
}

.memorial-text strong {
  color: #fedd00;
  font-weight: 600;
}

.site-footer {
  background: #141414;
  border-top: 1px solid #2a2a2a;
  padding: 1.5rem;
  margin-top: auto;
  width: 100%;
}

.footer-content {
  max-width: 720px;
  margin: 0 auto;
  display: flex;
  justify-content: center;
}

.footer-memorial {
  font-family: 'DM Mono', monospace;
  font-size: 0.8rem;
  color: #aaa;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.footer-memorial a {
  color: #fedd00;
  text-decoration: none;
  transition: opacity 0.15s;
}

.footer-memorial a:hover {
  opacity: 0.8;
}

.memorial-icon-small {
  font-size: 0.9rem;
  opacity: 0.7;
}

@media (max-width: 600px) {
  .memorial-banner {
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.4rem 0.75rem;
  }

  .memorial-text {
    font-size: 0.7rem;
    text-align: center;
  }

  .site-footer {
    padding: 1rem;
  }

  .footer-memorial {
    font-size: 0.7rem;
    flex-direction: column;
    gap: 0.3rem;
    text-align: center;
  }
}
</style>
