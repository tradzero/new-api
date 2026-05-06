import { createContext, useContext, useEffect, useState } from 'react'
import { getCookie, removeCookie, setCookie } from '@/lib/cookies'

/**
 * Theme presets — value `default` means "no preset", i.e. fall back to the
 * `:root` palette defined in `theme.css`. Other values map 1:1 to
 * `[data-theme-preset="..."]` selectors in CSS.
 */
export const THEME_PRESETS = [
  'default',
  'underground',
  'rose-garden',
  'lake-view',
  'sunset-glow',
  'forest-whisper',
  'ocean-breeze',
  'lavender-dream',
] as const
export type ThemePreset = (typeof THEME_PRESETS)[number]

/**
 * Border-radius scale — value `default` means "use whatever the preset (or
 * `:root`) says". Numeric values map 1:1 to `[data-theme-radius="..."]`
 * selectors in CSS (units are `rem`).
 */
export const THEME_RADII = ['default', '0', '0.3', '0.5', '0.75', '1'] as const
export type ThemeRadius = (typeof THEME_RADII)[number]

const DEFAULT_PRESET: ThemePreset = 'default'
const DEFAULT_RADIUS: ThemeRadius = 'default'

const PRESET_COOKIE_NAME = 'theme-preset'
const RADIUS_COOKIE_NAME = 'theme-radius'
const COOKIE_MAX_AGE = 60 * 60 * 24 * 365 // 1 year

const PRESET_ATTR = 'data-theme-preset'
const RADIUS_ATTR = 'data-theme-radius'

type ThemeCustomizationContextType = {
  defaultPreset: ThemePreset
  preset: ThemePreset
  setPreset: (preset: ThemePreset) => void

  defaultRadius: ThemeRadius
  radius: ThemeRadius
  setRadius: (radius: ThemeRadius) => void

  resetThemeCustomization: () => void
}

const ThemeCustomizationContext =
  createContext<ThemeCustomizationContextType | null>(null)

function isThemePreset(v: string | undefined): v is ThemePreset {
  return !!v && (THEME_PRESETS as readonly string[]).includes(v)
}

function isThemeRadius(v: string | undefined): v is ThemeRadius {
  return !!v && (THEME_RADII as readonly string[]).includes(v)
}

function applyAttribute(name: string, value: string, isDefault: boolean) {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  if (isDefault) {
    root.removeAttribute(name)
  } else {
    root.setAttribute(name, value)
  }
}

export function ThemeCustomizationProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const [preset, _setPreset] = useState<ThemePreset>(() => {
    const saved = getCookie(PRESET_COOKIE_NAME)
    return isThemePreset(saved) ? saved : DEFAULT_PRESET
  })

  const [radius, _setRadius] = useState<ThemeRadius>(() => {
    const saved = getCookie(RADIUS_COOKIE_NAME)
    return isThemeRadius(saved) ? saved : DEFAULT_RADIUS
  })

  useEffect(() => {
    applyAttribute(PRESET_ATTR, preset, preset === DEFAULT_PRESET)
  }, [preset])

  useEffect(() => {
    applyAttribute(RADIUS_ATTR, radius, radius === DEFAULT_RADIUS)
  }, [radius])

  const setPreset = (next: ThemePreset) => {
    _setPreset(next)
    if (next === DEFAULT_PRESET) {
      removeCookie(PRESET_COOKIE_NAME)
    } else {
      setCookie(PRESET_COOKIE_NAME, next, COOKIE_MAX_AGE)
    }
  }

  const setRadius = (next: ThemeRadius) => {
    _setRadius(next)
    if (next === DEFAULT_RADIUS) {
      removeCookie(RADIUS_COOKIE_NAME)
    } else {
      setCookie(RADIUS_COOKIE_NAME, next, COOKIE_MAX_AGE)
    }
  }

  const resetThemeCustomization = () => {
    setPreset(DEFAULT_PRESET)
    setRadius(DEFAULT_RADIUS)
  }

  return (
    <ThemeCustomizationContext
      value={{
        defaultPreset: DEFAULT_PRESET,
        preset,
        setPreset,
        defaultRadius: DEFAULT_RADIUS,
        radius,
        setRadius,
        resetThemeCustomization,
      }}
    >
      {children}
    </ThemeCustomizationContext>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useThemeCustomization() {
  const context = useContext(ThemeCustomizationContext)
  if (!context) {
    throw new Error(
      'useThemeCustomization must be used within a ThemeCustomizationProvider'
    )
  }
  return context
}
