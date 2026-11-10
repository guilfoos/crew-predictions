<template>
  <header class="site-header">
    <a
      class="brand"
      href="/"
    >
      <span class="brand-memorial">⚫</span>
      Crew Predictions
    </a>
    <nav class="site-nav">
      <a
        class="btn-ghost"
        href="/leaderboard"
        data-testid="nav-leaderboard"
      >Leaderboard</a>
      <a
        class="btn-ghost"
        href="/rules"
      >Rules</a>
      <template v-if="user">
        <a
          class="btn-ghost"
          :href="`/profile/${user.userID}`"
        >{{ user.handle }}</a>
        <a
          class="btn-ghost"
          href="/auth/logout"
        >Sign out</a>
      </template>
      <a
        v-else-if="!loading"
        class="btn-primary"
        href="/login"
      >Sign In</a>
    </nav>
    <span
      v-if="user && isMobile"
      class="mobile-user-handle"
    >{{ user.handle }}</span>
    <button
      class="hamburger"
      data-testid="hamburger"
      aria-label="Open menu"
      @click="drawerOpen = true"
    >
      <span /><span /><span />
    </button>
  </header>

  <template v-if="drawerOpen">
    <div
      class="drawer-backdrop"
      @click="drawerOpen = false"
    />
    <nav
      class="mobile-drawer"
      data-testid="mobile-drawer"
    >
      <a
        href="/leaderboard"
        data-testid="drawer-leaderboard"
        @click="drawerOpen = false"
      >Leaderboard</a>
      <a
        href="/rules"
        @click="drawerOpen = false"
      >Rules</a>
      <template v-if="user">
        <a
          :href="`/profile/${user.userID}`"
          @click="drawerOpen = false"
        >{{ user.handle }}</a>
        <a
          href="/auth/logout"
          @click="drawerOpen = false"
        >Sign out</a>
      </template>
      <a
        v-else-if="!loading"
        href="/login"
        @click="drawerOpen = false"
      >Sign In</a>
    </nav>
  </template>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

defineProps<{
  user: { userID: string; handle: string } | null
  loading?: boolean
}>()

const drawerOpen = ref(false)
const isMobile = ref(false)

function checkMobile() {
  isMobile.value = window.innerWidth <= 480
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>
