import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import zh from './zh.json'

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      zh: { translation: zh },
    },
    fallbackLng: 'zh',
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      lookupLocalStorage: 'ziziphus_language',
    },
  })

// Lazy-load non-zh language bundles on demand
let loading: string | null = null

i18n.on('languageChanged', (lng) => {
  if (lng === 'zh' || loading === lng) return
  if (i18n.hasResourceBundle(lng, 'translation')) return
  loading = lng
  import(`./${lng}.json`).then((mod) => {
    i18n.addResourceBundle(lng, 'translation', mod.default)
    loading = null
    i18n.changeLanguage(lng)
  })
})

export default i18n
