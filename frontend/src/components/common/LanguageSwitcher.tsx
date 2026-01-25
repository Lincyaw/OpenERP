/**
 * LanguageSwitcher Component
 *
 * A dropdown component for switching between supported languages.
 * Uses Semi Design Dropdown with language list from i18n config.
 */

import { Dropdown, Button } from '@douyinfe/semi-ui-19'
import { IconLanguage } from '@douyinfe/semi-icons'
import { useI18n } from '@/hooks'
import type { SupportedLanguage } from '@/i18n/config'

interface LanguageSwitcherProps {
  /** Show label next to icon */
  showLabel?: boolean
  /** Button size */
  size?: 'small' | 'default' | 'large'
  /** Additional class name */
  className?: string
}

/**
 * Language switcher dropdown component
 *
 * @example
 * ```tsx
 * // Icon only (default)
 * <LanguageSwitcher />
 *
 * // With label
 * <LanguageSwitcher showLabel />
 *
 * // Custom size
 * <LanguageSwitcher size="small" />
 * ```
 */
export function LanguageSwitcher({
  showLabel = false,
  size = 'default',
  className,
}: LanguageSwitcherProps) {
  const { language, changeLanguage, languages } = useI18n()

  const handleLanguageChange = (lng: SupportedLanguage) => {
    changeLanguage(lng)
  }

  const menuItems = languages.map((lang) => ({
    node: 'item' as const,
    key: lang.code,
    name: `${lang.flag} ${lang.name}`,
    active: lang.isCurrent,
    onClick: () => handleLanguageChange(lang.code),
  }))

  const currentLanguage = languages.find((l) => l.code === language)

  return (
    <Dropdown
      trigger="click"
      position="bottomRight"
      menu={menuItems}
      getPopupContainer={() => document.body}
    >
      <span style={{ display: 'inline-flex' }}>
        <Button
          theme="borderless"
          icon={<IconLanguage />}
          size={size}
          className={className}
          aria-label="Switch language"
        >
          {showLabel && currentLanguage && (
            <span style={{ marginLeft: '4px' }}>
              {currentLanguage.flag} {currentLanguage.name}
            </span>
          )}
        </Button>
      </span>
    </Dropdown>
  )
}

export default LanguageSwitcher
