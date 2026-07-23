import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  detectDesktopUpdate,
  onWailsEvent,
  openDesktopUpdateDialog,
  UPDATER_EVENTS,
  type UpdateCheckResult,
} from '../lib/checkUpdate'
import { isDesktop } from '../lib/desktop'

export const useUpdateStore = defineStore('updates', () => {
  const desktop = isDesktop()
  const available = ref(false)
  const current = ref('')
  const latest = ref('')
  const notes = ref('')
  const checking = ref(false)
  const lastError = ref('')
  const lastCheckedAt = ref<number | null>(null)
  let started = false
  const unsubs: Array<() => void> = []

  function applyResult(r: UpdateCheckResult) {
    current.value = r.current || current.value
    lastError.value = r.error || ''
    lastCheckedAt.value = Date.now()
    if (r.error) return
    if (r.available && r.latest) {
      available.value = true
      latest.value = r.latest
      notes.value = r.notes || ''
    } else {
      available.value = false
      latest.value = ''
      notes.value = ''
    }
  }

  async function detect() {
    if (!desktop) return null
    checking.value = true
    lastError.value = ''
    try {
      const r = await detectDesktopUpdate()
      if (r) applyResult(r)
      return r
    } finally {
      checking.value = false
    }
  }

  async function openDialog() {
    if (!desktop) return false
    return openDesktopUpdateDialog()
  }

  async function start() {
    if (!desktop || started) return
    started = true

    unsubs.push(
      await onWailsEvent(UPDATER_EVENTS.updateAvailable, (data) => {
        const v = data && typeof data === 'object'
          ? String((data as { version?: unknown }).version ?? '')
          : ''
        if (v) {
          available.value = true
          latest.value = v
          const n = data && typeof data === 'object'
            ? (data as { notes?: unknown }).notes
            : undefined
          notes.value = n != null ? String(n) : ''
          lastCheckedAt.value = Date.now()
        }
      }),
    )
    unsubs.push(
      await onWailsEvent(UPDATER_EVENTS.noUpdate, () => {
        available.value = false
        latest.value = ''
        notes.value = ''
        lastError.value = ''
        lastCheckedAt.value = Date.now()
      }),
    )

    // Quiet first check; Go also polls after ~45s.
    void detect()
  }

  function stop() {
    for (const off of unsubs) off()
    unsubs.length = 0
    started = false
  }

  return {
    desktop,
    available,
    current,
    latest,
    notes,
    checking,
    lastError,
    lastCheckedAt,
    detect,
    openDialog,
    start,
    stop,
  }
})
