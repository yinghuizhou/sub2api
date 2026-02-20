<template>
  <div
    class="flex min-h-screen items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 p-4 dark:from-dark-900 dark:to-dark-800"
  >
    <div class="w-full max-w-2xl">
      <!-- Logo & Title -->
      <div class="mb-8 text-center">
        <div
          class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-500 to-primary-600 shadow-lg"
        >
          <Icon name="cog" size="xl" class="text-white" />
        </div>
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">
          {{ isAdminOnly ? t('setup.welcome.title') : t('setup.title') }}
        </h1>
        <p class="mt-2 text-gray-500 dark:text-dark-400">
          {{ isAdminOnly ? t('setup.welcome.description') : t('setup.description') }}
        </p>
      </div>

      <!-- Progress Steps (hidden in admin-only mode with single visible step) -->
      <div v-if="visibleSteps.length > 1" class="mb-8">
        <div class="flex items-center justify-center">
          <template v-for="(step, index) in visibleSteps" :key="step.id">
            <div class="flex items-center">
              <div
                :class="[
                  'flex h-10 w-10 items-center justify-center rounded-full text-sm font-semibold transition-all',
                  currentVisibleStep > index
                    ? 'bg-primary-500 text-white'
                    : currentVisibleStep === index
                      ? 'bg-primary-500 text-white ring-4 ring-primary-100 dark:ring-primary-900'
                      : 'bg-gray-200 text-gray-500 dark:bg-dark-700 dark:text-dark-400'
                ]"
              >
                <Icon
                  v-if="currentVisibleStep > index"
                  name="check"
                  size="md"
                  :stroke-width="2"
                />
                <span v-else>{{ index + 1 }}</span>
              </div>
              <span
                class="ml-2 text-sm font-medium"
                :class="
                  currentVisibleStep >= index
                    ? 'text-gray-900 dark:text-white'
                    : 'text-gray-400 dark:text-dark-500'
                "
              >
                {{ step.title }}
              </span>
            </div>
            <div
              v-if="index < visibleSteps.length - 1"
              class="mx-3 h-0.5 w-12"
              :class="currentVisibleStep > index ? 'bg-primary-500' : 'bg-gray-200 dark:bg-dark-700'"
            ></div>
          </template>
        </div>
      </div>

      <!-- Step Content -->
      <div class="rounded-2xl bg-white p-8 shadow-xl dark:bg-dark-800">
        <!-- Database Step -->
        <div v-if="activeStepId === 'database'" class="space-y-6">
          <div class="mb-6 text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('setup.database.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('setup.database.description') }}
            </p>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="input-label">{{ t('setup.database.host') }}</label>
              <input v-model="dbForm.host" type="text" class="input" placeholder="localhost" />
            </div>
            <div>
              <label class="input-label">{{ t('setup.database.port') }}</label>
              <input v-model.number="dbForm.port" type="number" class="input" placeholder="5432" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="input-label">{{ t('setup.database.username') }}</label>
              <input v-model="dbForm.user" type="text" class="input" placeholder="postgres" />
            </div>
            <div>
              <label class="input-label">{{ t('setup.database.password') }}</label>
              <input
                v-model="dbForm.password"
                type="password"
                class="input"
                :placeholder="t('setup.database.passwordPlaceholder')"
              />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="input-label">{{ t('setup.database.databaseName') }}</label>
              <input v-model="dbForm.dbname" type="text" class="input" placeholder="sub2api" />
            </div>
            <div>
              <label class="input-label">{{ t('setup.database.sslMode') }}</label>
              <Select
                v-model="dbForm.sslmode"
                :options="[
                  { value: 'disable', label: t('setup.database.ssl.disable') },
                  { value: 'require', label: t('setup.database.ssl.require') },
                  { value: 'verify-ca', label: t('setup.database.ssl.verifyCa') },
                  { value: 'verify-full', label: t('setup.database.ssl.verifyFull') }
                ]"
              />
            </div>
          </div>

          <button @click="testDatabaseConnection" :disabled="testingDb" class="btn btn-secondary w-full">
            <svg v-if="testingDb" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <Icon v-else-if="dbConnected" name="check" size="md" class="mr-2 text-green-500" :stroke-width="2" />
            {{ testingDb ? t('setup.status.testing') : dbConnected ? t('setup.status.success') : t('setup.status.testConnection') }}
          </button>
        </div>

        <!-- Redis Step -->
        <div v-if="activeStepId === 'redis'" class="space-y-6">
          <div class="mb-6 text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('setup.redis.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('setup.redis.description') }}
            </p>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="input-label">{{ t('setup.redis.host') }}</label>
              <input v-model="redisForm.host" type="text" class="input" placeholder="localhost" />
            </div>
            <div>
              <label class="input-label">{{ t('setup.redis.port') }}</label>
              <input v-model.number="redisForm.port" type="number" class="input" placeholder="6379" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="input-label">{{ t('setup.redis.password') }}</label>
              <input
                v-model="redisForm.password"
                type="password"
                class="input"
                :placeholder="t('setup.redis.passwordPlaceholder')"
              />
            </div>
            <div>
              <label class="input-label">{{ t('setup.redis.database') }}</label>
              <input v-model.number="redisForm.db" type="number" class="input" placeholder="0" />
            </div>
          </div>

          <div class="flex items-center justify-between rounded-xl border border-gray-200 p-3 dark:border-dark-700">
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('setup.redis.enableTls') }}</p>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('setup.redis.enableTlsHint') }}</p>
            </div>
            <Toggle v-model="redisForm.enable_tls" />
          </div>

          <button @click="testRedisConnection" :disabled="testingRedis" class="btn btn-secondary w-full">
            <svg v-if="testingRedis" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <Icon v-else-if="redisConnected" name="check" size="md" class="mr-2 text-green-500" :stroke-width="2" />
            {{ testingRedis ? t('setup.status.testing') : redisConnected ? t('setup.status.success') : t('setup.status.testConnection') }}
          </button>
        </div>

        <!-- Admin Step -->
        <div v-if="activeStepId === 'admin'" class="space-y-6">
          <div class="mb-6 text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('setup.admin.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('setup.admin.description') }}
            </p>
          </div>

          <div>
            <label class="input-label">{{ t('setup.admin.email') }}</label>
            <input v-model="adminForm.email" type="email" class="input" placeholder="admin@example.com" />
          </div>

          <div>
            <label class="input-label">{{ t('setup.admin.password') }}</label>
            <input
              v-model="adminForm.password"
              type="password"
              class="input"
              :placeholder="t('setup.admin.passwordPlaceholder')"
            />
          </div>

          <div>
            <label class="input-label">{{ t('setup.admin.confirmPassword') }}</label>
            <input
              v-model="confirmPassword"
              type="password"
              class="input"
              :placeholder="t('setup.admin.confirmPasswordPlaceholder')"
            />
            <p
              v-if="confirmPassword && adminForm.password !== confirmPassword"
              class="input-error-text"
            >
              {{ t('setup.admin.passwordMismatch') }}
            </p>
          </div>
        </div>

        <!-- Complete/Review Step -->
        <div v-if="activeStepId === 'complete'" class="space-y-6">
          <div class="mb-6 text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('setup.ready.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('setup.ready.description') }}
            </p>
          </div>

          <div class="space-y-4">
            <div v-if="!isStepConfigured('database')" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('setup.ready.database') }}
              </h3>
              <p class="text-gray-900 dark:text-white">
                {{ dbForm.user }}@{{ dbForm.host }}:{{ dbForm.port }}/{{ dbForm.dbname }}
              </p>
            </div>

            <div v-if="!isStepConfigured('redis')" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('setup.ready.redis') }}
              </h3>
              <p class="text-gray-900 dark:text-white">
                {{ redisForm.host }}:{{ redisForm.port }}
              </p>
            </div>

            <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('setup.ready.adminEmail') }}
              </h3>
              <p class="text-gray-900 dark:text-white">{{ adminForm.email }}</p>
            </div>
          </div>
        </div>

        <!-- Error Message -->
        <div
          v-if="errorMessage"
          class="mt-6 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
        >
          <div class="flex items-start gap-3">
            <Icon name="exclamationCircle" size="md" class="flex-shrink-0 text-red-500" />
            <p class="text-sm text-red-700 dark:text-red-400">{{ errorMessage }}</p>
          </div>
        </div>

        <!-- Success Message -->
        <div
          v-if="installSuccess"
          class="mt-6 rounded-xl border border-green-200 bg-green-50 p-4 dark:border-green-800/50 dark:bg-green-900/20"
        >
          <div class="flex items-start gap-3">
            <svg
              v-if="!serviceReady"
              class="h-5 w-5 flex-shrink-0 animate-spin text-green-500"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <Icon v-else name="checkCircle" size="md" class="flex-shrink-0 text-green-500" />
            <div>
              <p class="text-sm font-medium text-green-700 dark:text-green-400">
                {{ t('setup.status.completed') }}
              </p>
              <p class="mt-1 text-sm text-green-600 dark:text-green-500">
                {{ serviceReady ? t('setup.status.redirecting') : t('setup.status.restarting') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Navigation Buttons -->
        <div class="mt-8 flex justify-between">
          <button
            v-if="currentVisibleStep > 0 && !installSuccess"
            @click="currentVisibleStep--"
            class="btn btn-secondary"
          >
            <Icon name="chevronLeft" size="sm" class="mr-2" :stroke-width="2" />
            {{ t('common.back') }}
          </button>
          <div v-else></div>

          <!-- Next button (not on last step) -->
          <button
            v-if="!isLastVisibleStep"
            @click="nextStep"
            :disabled="!canProceed"
            class="btn btn-primary"
          >
            <!-- In admin-only mode, admin step directly leads to install -->
            {{ isAdminOnly && activeStepId === 'admin'
              ? (installing ? t('setup.status.installing') : t('setup.status.completeInstallation'))
              : t('common.next') }}
            <Icon v-if="!(isAdminOnly && activeStepId === 'admin')" name="chevronRight" size="sm" class="ml-2" :stroke-width="2" />
          </button>

          <!-- Install button (last step = complete/review) -->
          <button
            v-else-if="!installSuccess"
            @click="performInstall"
            :disabled="installing"
            class="btn btn-primary"
          >
            <svg v-if="installing" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ installing ? t('setup.status.installing') : t('setup.status.completeInstallation') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getSetupStatus,
  testDatabase,
  testRedis,
  install,
  type DatabaseConfig,
  type RedisConfig,
  type InstallRequest,
} from '@/api/setup'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

// Pre-configured steps from backend (e.g. ["database", "redis"])
const configuredSteps = ref<string[]>([])
const loading = ref(true)

// All possible steps
const allSteps = computed(() => [
  { id: 'database', title: t('setup.database.title') },
  { id: 'redis', title: t('setup.redis.title') },
  { id: 'admin', title: t('setup.admin.title') },
  { id: 'complete', title: t('setup.ready.title') },
])

// Steps visible to the user (skip pre-configured infra steps)
const visibleSteps = computed(() =>
  allSteps.value.filter(
    (step) => !configuredSteps.value.includes(step.id),
  ),
)

// True when only admin + complete steps are visible (Docker with env vars)
const isAdminOnly = computed(() => {
  const ids = visibleSteps.value.map((s) => s.id)
  return ids.length <= 2 && ids.includes('admin')
})

const currentVisibleStep = ref(0)
const activeStepId = computed(() => visibleSteps.value[currentVisibleStep.value]?.id ?? 'admin')
const isLastVisibleStep = computed(() => currentVisibleStep.value === visibleSteps.value.length - 1)

const errorMessage = ref('')
const installSuccess = ref(false)
const installing = ref(false)
const confirmPassword = ref('')
const serviceReady = ref(false)

// Connection test states
const testingDb = ref(false)
const testingRedis = ref(false)
const dbConnected = ref(false)
const redisConnected = ref(false)

function isStepConfigured(stepId: string): boolean {
  return configuredSteps.value.includes(stepId)
}

// Form data (separate reactive objects for each section)
const dbForm = reactive<DatabaseConfig>({
  host: 'localhost',
  port: 5432,
  user: 'postgres',
  password: '',
  dbname: 'sub2api',
  sslmode: 'disable',
})

const redisForm = reactive<RedisConfig>({
  host: 'localhost',
  port: 6379,
  password: '',
  db: 0,
  enable_tls: false,
})

const adminForm = reactive({
  email: '',
  password: '',
})

const canProceed = computed(() => {
  switch (activeStepId.value) {
    case 'database':
      return dbConnected.value
    case 'redis':
      return redisConnected.value
    case 'admin':
      return (
        adminForm.email.length > 0 &&
        adminForm.password.length >= 6 &&
        adminForm.password === confirmPassword.value
      )
    default:
      return true
  }
})

// Fetch setup status on mount
onMounted(async () => {
  try {
    const status = await getSetupStatus()
    configuredSteps.value = status.configured_steps || []
  } catch {
    // If fetch fails, show all steps
    configuredSteps.value = []
  } finally {
    loading.value = false
  }
})

async function testDatabaseConnection() {
  testingDb.value = true
  errorMessage.value = ''
  dbConnected.value = false

  try {
    await testDatabase(dbForm)
    dbConnected.value = true
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string } }; message?: string }
    errorMessage.value = err.response?.data?.detail || err.message || 'Connection failed'
  } finally {
    testingDb.value = false
  }
}

async function testRedisConnection() {
  testingRedis.value = true
  errorMessage.value = ''
  redisConnected.value = false

  try {
    await testRedis(redisForm)
    redisConnected.value = true
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string } }; message?: string }
    errorMessage.value = err.response?.data?.detail || err.message || 'Connection failed'
  } finally {
    testingRedis.value = false
  }
}

function nextStep() {
  if (!canProceed.value) return
  errorMessage.value = ''

  // In admin-only mode, skip the review step and install directly from admin step
  if (isAdminOnly.value && activeStepId.value === 'admin') {
    performInstall()
    return
  }

  currentVisibleStep.value++
}

function buildInstallPayload(): InstallRequest {
  const payload: InstallRequest = {
    admin: { ...adminForm },
  }

  // Only include DB/Redis config if they were NOT pre-configured via env
  if (!isStepConfigured('database')) {
    payload.database = { ...dbForm }
  }
  if (!isStepConfigured('redis')) {
    payload.redis = { ...redisForm }
  }

  return payload
}

async function performInstall() {
  installing.value = true
  errorMessage.value = ''

  try {
    await install(buildInstallPayload())
    installSuccess.value = true
    waitForServiceRestart()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string } }; message?: string }
    errorMessage.value = err.response?.data?.detail || err.message || 'Installation failed'
  } finally {
    installing.value = false
  }
}

async function waitForServiceRestart() {
  const maxAttempts = 60
  const interval = 1000

  await new Promise((resolve) => setTimeout(resolve, 3000))

  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      const response = await fetch('/setup/status', { method: 'GET', cache: 'no-store' })
      if (response.ok) {
        const data = await response.json()
        if (data.data && !data.data.needs_setup) {
          serviceReady.value = true
          setTimeout(() => {
            window.location.href = '/login'
          }, 1500)
          return
        }
      }
    } catch {
      // Service not ready, continue polling
    }
    await new Promise((resolve) => setTimeout(resolve, interval))
  }

  errorMessage.value = t('setup.status.timeout')
}
</script>
