<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import { SUPPORTED_LOCALES, setLocale, type AppLocale } from '../i18n'

const { t, locale } = useI18n()

const searchProvider = ref('duckduckgo')
const searchBaseURL = ref('')
const searchAPIKey = ref('')
const searchAPIKeySet = ref(false)
const searchSaving = ref(false)
const searchError = ref('')
const searchSaved = ref(false)

const searchProviders = computed(() => [
  { value: '', label: t('settings.providers.disabled') },
  { value: 'bing', label: t('settings.providers.bing') },
  { value: 'google', label: t('settings.providers.google') },
  { value: 'duckduckgo', label: t('settings.providers.duckduckgo') },
  { value: 'brave', label: t('settings.providers.brave') },
  { value: 'searxng', label: t('settings.providers.searxng') },
])

const currentLocale = computed({
  get: () => (locale.value === 'en' ? 'en' : 'zh-CN') as AppLocale,
  set: (v: AppLocale) => setLocale(v),
})

onMounted(() => {
  void loadSearchSettings()
})

async function loadSearchSettings() {
  searchError.value = ''
  try {
    const r = await api.getSearchSettings()
    searchProvider.value = r.provider ?? ''
    searchBaseURL.value = r.base_url ?? ''
    searchAPIKeySet.value = !!r.api_key_set
    searchAPIKey.value = ''
  } catch (e: unknown) {
    searchError.value = e instanceof Error ? e.message : t('common.failedToLoad')
  }
}

async function saveSearchSettings() {
  searchSaving.value = true
  searchError.value = ''
  searchSaved.value = false
  try {
    const body: { provider?: string; api_key?: string; base_url?: string } = {
      provider: searchProvider.value,
      base_url: searchBaseURL.value,
    }
    if (searchAPIKey.value.trim()) {
      body.api_key = searchAPIKey.value.trim()
    }
    const r = await api.putSearchSettings(body)
    searchProvider.value = r.provider ?? searchProvider.value
    searchBaseURL.value = r.base_url ?? searchBaseURL.value
    searchAPIKeySet.value = !!r.api_key_set
    searchAPIKey.value = ''
    searchSaved.value = true
  } catch (e: unknown) {
    searchError.value = e instanceof Error ? e.message : t('common.saveFailed')
  } finally {
    searchSaving.value = false
  }
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto space-y-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold mb-1">{{ t('settings.title') }}</h1>
        <p class="text-sm text-neutral-500">{{ t('settings.subtitle') }}</p>
      </div>
    </div>

    <div v-if="searchError" class="text-sm text-red-600">{{ searchError }}</div>

    <!-- Language -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div>
        <h2 class="text-sm font-semibold text-neutral-900">{{ t('settings.language') }}</h2>
        <p class="text-xs text-neutral-500 mt-0.5">{{ t('settings.languageHint') }}</p>
      </div>
      <label class="block">
        <select
          v-model="currentLocale"
          class="w-full max-w-xs border border-neutral-200 rounded px-2 py-1.5 text-sm bg-white"
        >
          <option v-for="opt in SUPPORTED_LOCALES" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
      </label>
    </section>

    <!-- Web search -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-neutral-900">{{ t('settings.searchTitle') }}</h2>
          <p class="text-xs text-neutral-500 mt-0.5">
            {{ t('settings.searchHint') }}
          </p>
        </div>
        <button
          type="button"
          class="shrink-0 h-8 px-3 rounded bg-neutral-800 text-white text-sm hover:bg-neutral-700 disabled:opacity-50"
          :disabled="searchSaving"
          @click="saveSearchSettings"
        >{{ searchSaving ? t('common.saving') : t('common.save') }}</button>
      </div>

      <label class="block">
        <span class="block text-xs text-neutral-500 mb-1">{{ t('settings.provider') }}</span>
        <select
          v-model="searchProvider"
          class="w-full border border-neutral-200 rounded px-2 py-1.5 text-sm bg-white"
        >
          <option v-for="p in searchProviders" :key="p.value || 'off'" :value="p.value">
            {{ p.label }}
          </option>
        </select>
      </label>

      <label v-if="searchProvider === 'brave'" class="block">
        <span class="block text-xs text-neutral-500 mb-1">
          {{ t('settings.braveKey') }}
          <span v-if="searchAPIKeySet" class="text-neutral-400">{{ t('settings.braveKeySaved') }}</span>
        </span>
        <input
          v-model="searchAPIKey"
          type="password"
          autocomplete="off"
          placeholder="BSA..."
          class="w-full border border-neutral-200 rounded px-2 py-1.5 text-sm"
        />
      </label>

      <label v-if="searchProvider === 'searxng'" class="block">
        <span class="block text-xs text-neutral-500 mb-1">{{ t('settings.searxngUrl') }}</span>
        <input
          v-model="searchBaseURL"
          type="url"
          placeholder="https://searx.example"
          class="w-full border border-neutral-200 rounded px-2 py-1.5 text-sm"
        />
      </label>

      <p
        v-if="searchProvider === 'bing' || searchProvider === 'google'"
        class="text-xs text-neutral-500 leading-relaxed"
      >
        {{ t('settings.browserSearchHint') }}
      </p>

      <p v-if="searchSaved" class="text-xs text-green-700">{{ t('common.saved') }}</p>
    </section>
  </div>
</template>
