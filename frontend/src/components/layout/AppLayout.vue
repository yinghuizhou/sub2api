<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <!-- Background Decoration -->
    <div class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="p-4 md:p-6 lg:p-8">
        <div v-if="renderError" class="flex flex-col items-center justify-center py-20 text-center">
          <div class="mb-4 text-5xl">⚠️</div>
          <h2 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">页面加载出错</h2>
          <p class="mb-6 text-sm text-gray-500 dark:text-gray-400">{{ renderError }}</p>
          <button @click="handleRetry" class="btn btn-primary">刷新页面</button>
        </div>
        <slot v-else />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted, onErrorCaptured, ref } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

const renderError = ref<string | null>(null)

onErrorCaptured((err) => {
  console.error('[AppLayout] Child component error:', err)
  renderError.value = err instanceof Error ? err.message : String(err)
  return false // prevent further propagation
})

const handleRetry = () => {
  renderError.value = null
  window.location.reload()
}

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
