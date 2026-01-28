/**
 * i18n Initialization
 *
 * This file initializes i18next with all necessary plugins and configuration.
 * It supports:
 * - Chinese (zh-CN) and English (en-US)
 * - Browser language detection
 * - Lazy loading of translations
 * - Namespace separation
 */

import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

import {
  DEFAULT_LANGUAGE,
  FALLBACK_LANGUAGE,
  SUPPORTED_LANGUAGES,
  DEFAULT_NAMESPACE,
  NAMESPACES,
} from './config'

// Import translation files directly for bundling
// This approach ensures translations are bundled with the app
import zhCNCommon from '../locales/zh-CN/common.json'
import enUSCommon from '../locales/en-US/common.json'
import zhCNValidation from '../locales/zh-CN/validation.json'
import enUSValidation from '../locales/en-US/validation.json'
import zhCNAuth from '../locales/zh-CN/auth.json'
import enUSAuth from '../locales/en-US/auth.json'
import zhCNCatalog from '../locales/zh-CN/catalog.json'
import enUSCatalog from '../locales/en-US/catalog.json'
import zhCNPartner from '../locales/zh-CN/partner.json'
import enUSPartner from '../locales/en-US/partner.json'
import zhCNInventory from '../locales/zh-CN/inventory.json'
import enUSInventory from '../locales/en-US/inventory.json'
import zhCNTrade from '../locales/zh-CN/trade.json'
import enUSTrade from '../locales/en-US/trade.json'
import zhCNFinance from '../locales/zh-CN/finance.json'
import enUSFinance from '../locales/en-US/finance.json'
import zhCNSystem from '../locales/zh-CN/system.json'
import enUSSystem from '../locales/en-US/system.json'
import zhCNIntegration from '../locales/zh-CN/integration.json'
import enUSIntegration from '../locales/en-US/integration.json'
import zhCNAdmin from '../locales/zh-CN/admin.json'
import enUSAdmin from '../locales/en-US/admin.json'

/**
 * Translation resources
 *
 * Using inline imports for better bundling and avoiding HTTP requests.
 * Additional namespaces will be added as they are implemented.
 */
const resources = {
  'zh-CN': {
    common: zhCNCommon,
    validation: zhCNValidation,
    auth: zhCNAuth,
    catalog: zhCNCatalog,
    partner: zhCNPartner,
    inventory: zhCNInventory,
    trade: zhCNTrade,
    finance: zhCNFinance,
    system: zhCNSystem,
    integration: zhCNIntegration,
    admin: zhCNAdmin,
  },
  'en-US': {
    common: enUSCommon,
    validation: enUSValidation,
    auth: enUSAuth,
    catalog: enUSCatalog,
    partner: enUSPartner,
    inventory: enUSInventory,
    trade: enUSTrade,
    finance: enUSFinance,
    system: enUSSystem,
    integration: enUSIntegration,
    admin: enUSAdmin,
  },
}

/**
 * Initialize i18next
 */
i18n
  // Detect user language from browser
  .use(LanguageDetector)
  // Pass the i18n instance to react-i18next
  .use(initReactI18next)
  // Initialize with configuration
  .init({
    resources,
    lng: DEFAULT_LANGUAGE,
    fallbackLng: FALLBACK_LANGUAGE,
    supportedLngs: SUPPORTED_LANGUAGES as unknown as string[],
    defaultNS: DEFAULT_NAMESPACE,
    ns: NAMESPACES as unknown as string[],

    // Only load the current language, don't try to load language hierarchy (zh-CN -> zh)
    load: 'currentOnly',

    // Detection options
    detection: {
      // Order of language detection methods
      order: ['localStorage', 'navigator', 'htmlTag'],
      // Cache language in localStorage
      caches: ['localStorage'],
      // Key used in localStorage
      lookupLocalStorage: 'erp-language',
      // Convert language codes like 'zh' to 'zh-CN'
      convertDetectedLanguage: (lng: string) => {
        // Map short codes to full codes
        const languageMap: Record<string, string> = {
          zh: 'zh-CN',
          en: 'en-US',
        }
        return languageMap[lng] || lng
      },
    },

    // Interpolation settings
    interpolation: {
      // React already handles XSS
      escapeValue: false,
      // Format function for custom formatting
      format: (value, format, lng) => {
        if (format === 'number' && typeof value === 'number') {
          return new Intl.NumberFormat(lng).format(value)
        }
        if (format === 'currency' && typeof value === 'number') {
          return new Intl.NumberFormat(lng, {
            style: 'currency',
            currency: lng === 'zh-CN' ? 'CNY' : 'USD',
          }).format(value)
        }
        if (format === 'date' && value instanceof Date) {
          return new Intl.DateTimeFormat(lng).format(value)
        }
        return String(value)
      },
    },

    // React specific options
    react: {
      // Wait for translations to be loaded before rendering
      useSuspense: true,
      // Bind i18n store to React
      bindI18n: 'languageChanged loaded',
      // Bind i18n store to React on remove
      bindI18nStore: 'added removed',
    },

    // Debug mode (disable in production)
    debug: import.meta.env.DEV,

    // Key separator for nested translations
    keySeparator: '.',

    // Namespace separator
    nsSeparator: ':',

    // Return empty string for missing keys in production
    returnEmptyString: false,

    // Return key for missing translations in development
    returnNull: false,
  })

export default i18n
