<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'

const searchProvider = ref('duckduckgo')
const searchBaseURL = ref('')
const searchAPIKey = ref('')
const searchAPIKeySet = ref(false)
const searchSaving = ref(false)
const searchError = ref('')
const searchSaved = ref(false)

const searchProviders = [
  { value: '', label: 'Disabled' },
  { value: 'bing', label: 'Bing（浏览器）' },
  { value: 'google', label: 'Google（浏览器）' },
  { value: 'duckduckgo', label: 'DuckDuckGo' },
  { value: 'brave', label: 'Brave API' },
  { value: 'searxng', label: 'SearXNG' },
] as const

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
    searchError.value = e instanceof Error ? e.message : 'failed to load'
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
    searchError.value = e instanceof Error ? e.message : 'save failed'
  } finally {
    searchSaving.value = false
  }
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto space-y-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold mb-1">Setting</h1>
        <p class="text-sm text-neutral-500">系统级配置：搜索引擎等。</p>
      </div>
    </div>

    <div v-if="searchError" class="text-sm text-red-600">{{ searchError }}</div>

    <!-- Web search -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-neutral-900">Web search</h2>
          <p class="text-xs text-neutral-500 mt-0.5">
            供 <code class="bg-neutral-100 px-1 py-0.5 rounded">web_search</code> 使用的搜索后端。
          </p>
        </div>
        <button
          type="button"
          class="shrink-0 h-8 px-3 rounded bg-neutral-800 text-white text-sm hover:bg-neutral-700 disabled:opacity-50"
          :disabled="searchSaving"
          @click="saveSearchSettings"
        >{{ searchSaving ? '保存中…' : '保存' }}</button>
      </div>

      <label class="block">
        <span class="block text-xs text-neutral-500 mb-1">Provider</span>
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
          Brave API key
          <span v-if="searchAPIKeySet" class="text-neutral-400">（已保存，留空不修改）</span>
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
        <span class="block text-xs text-neutral-500 mb-1">SearXNG base URL</span>
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
        通过内置浏览器打开搜索结果页并解析链接，需在 Tools 中启用 Browser。大陆环境建议选 Bing。
      </p>

      <p v-if="searchSaved" class="text-xs text-green-700">已保存</p>
    </section>
  </div>
</template>
