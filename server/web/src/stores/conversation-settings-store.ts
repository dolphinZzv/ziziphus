import { getItem, setItem } from '@/lib/storage'

export interface ConvSettings {
  showAgentResponseOnly: boolean
}

const STORAGE_KEY = 'conv_settings'

type Listener = () => void
const listeners = new Set<Listener>()

function loadAll(): Record<string, ConvSettings> {
  return getItem<Record<string, ConvSettings>>(STORAGE_KEY, {}) || {}
}

function saveAll(all: Record<string, ConvSettings>) {
  setItem(STORAGE_KEY, all)
}

function emit() {
  listeners.forEach(fn => fn())
}

export function getConvSettings(convId: string): ConvSettings {
  const all = loadAll()
  return all[convId] || { showAgentResponseOnly: false }
}

export function setConvSetting(convId: string, key: keyof ConvSettings, value: boolean) {
  const all = loadAll()
  all[convId] = { ...(all[convId] || {}), [key]: value }
  saveAll(all)
  emit()
}

export function toggleConvSetting(convId: string, key: keyof ConvSettings) {
  const current = getConvSettings(convId)
  setConvSetting(convId, key, !current[key])
}

export function subscribe(fn: Listener) {
  listeners.add(fn)
  return () => listeners.delete(fn)
}
