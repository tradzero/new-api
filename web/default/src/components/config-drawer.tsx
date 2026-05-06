import { type SVGProps } from 'react'
import { Root as Radio, Item } from '@radix-ui/react-radio-group'
import { CircleCheck, Palette, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { IconDir } from '@/assets/custom/icon-dir'
import { IconLayoutCompact } from '@/assets/custom/icon-layout-compact'
import { IconLayoutDefault } from '@/assets/custom/icon-layout-default'
import { IconLayoutFull } from '@/assets/custom/icon-layout-full'
import { IconSidebarFloating } from '@/assets/custom/icon-sidebar-floating'
import { IconSidebarInset } from '@/assets/custom/icon-sidebar-inset'
import { IconSidebarSidebar } from '@/assets/custom/icon-sidebar-sidebar'
import { IconThemeDark } from '@/assets/custom/icon-theme-dark'
import { IconThemeLight } from '@/assets/custom/icon-theme-light'
import { IconThemeSystem } from '@/assets/custom/icon-theme-system'
import { cn } from '@/lib/utils'
import { useDirection } from '@/context/direction-provider'
import { type Collapsible, useLayout } from '@/context/layout-provider'
import {
  type ThemePreset,
  type ThemeRadius,
  useThemeCustomization,
} from '@/context/theme-customization-provider'
import { useTheme } from '@/context/theme-provider'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { useSidebar } from './ui/sidebar'

export function ConfigDrawer() {
  const { t } = useTranslation()
  const { setOpen } = useSidebar()
  const { resetDir } = useDirection()
  const { resetTheme } = useTheme()
  const { resetThemeCustomization } = useThemeCustomization()
  const { resetLayout } = useLayout()

  const handleReset = () => {
    setOpen(true)
    resetDir()
    resetTheme()
    resetThemeCustomization()
    resetLayout()
  }

  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button
          size='icon'
          variant='ghost'
          aria-label={t('Open theme settings')}
          aria-describedby='config-drawer-description'
          className='rounded-full max-md:hidden'
        >
          <Palette className='size-[1.2rem]' aria-hidden='true' />
        </Button>
      </SheetTrigger>
      <SheetContent className='flex w-full flex-col sm:max-w-md'>
        <SheetHeader className='pb-0 text-start'>
          <SheetTitle>{t('Theme Settings')}</SheetTitle>
          <SheetDescription id='config-drawer-description'>
            {t('Adjust the appearance and layout to suit your preferences.')}
          </SheetDescription>
        </SheetHeader>
        <div className='space-y-6 overflow-y-auto px-4'>
          <ThemeConfig />
          <ThemePresetConfig />
          <RadiusConfig />
          <SidebarConfig />
          <LayoutConfig />
          <DirConfig />
        </div>
        <SheetFooter className='gap-2'>
          <Button
            variant='destructive'
            onClick={handleReset}
            aria-label={t('Reset all settings to default values')}
          >
            {t('Reset')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

function SectionTitle(props: {
  title: string
  showReset?: boolean
  onReset?: () => void
  className?: string
}) {
  return (
    <div
      className={cn(
        'text-muted-foreground mb-2 flex items-center gap-2 text-sm font-semibold',
        props.className
      )}
    >
      {props.title}
      {props.showReset && props.onReset && (
        <Button
          size='icon'
          variant='secondary'
          className='size-4 rounded-full'
          onClick={props.onReset}
          aria-label='Reset'
        >
          <RotateCcw className='size-3' aria-hidden='true' />
        </Button>
      )}
    </div>
  )
}

function RadioGroupItem(props: {
  item: {
    value: string
    label: string
    icon: (props: SVGProps<SVGSVGElement>) => React.ReactElement
  }
  isTheme?: boolean
}) {
  const isTheme = props.isTheme ?? false
  return (
    <Item
      value={props.item.value}
      className={cn('group outline-none', 'transition duration-200 ease-in')}
      aria-label={`Select ${props.item.label.toLowerCase()}`}
      aria-describedby={`${props.item.value}-description`}
    >
      <div
        className={cn(
          'ring-border relative rounded-[6px] ring-[1px]',
          'group-data-[state=checked]:ring-primary group-data-[state=checked]:shadow-2xl',
          'group-focus-visible:ring-2'
        )}
        role='img'
        aria-hidden='false'
        aria-label={`${props.item.label} option preview`}
      >
        <CircleCheck
          className={cn(
            'fill-primary size-6 stroke-white',
            'group-data-[state=unchecked]:hidden',
            'absolute top-0 right-0 translate-x-1/2 -translate-y-1/2'
          )}
          aria-hidden='true'
        />
        <props.item.icon
          className={cn(
            !isTheme &&
              'stroke-primary fill-primary group-data-[state=unchecked]:stroke-muted-foreground group-data-[state=unchecked]:fill-muted-foreground'
          )}
          aria-hidden='true'
        />
      </div>
      <div
        className='mt-1 text-xs'
        id={`${props.item.value}-description`}
        aria-live='polite'
      >
        {props.item.label}
      </div>
    </Item>
  )
}

function ThemeConfig() {
  const { t } = useTranslation()
  const { defaultTheme, theme, setTheme } = useTheme()
  return (
    <div>
      <SectionTitle
        title={t('Theme')}
        showReset={theme !== defaultTheme}
        onReset={() => setTheme(defaultTheme)}
      />
      <Radio
        value={theme}
        onValueChange={setTheme}
        className='grid w-full max-w-md grid-cols-3 gap-4'
        aria-label={t('Select theme preference')}
        aria-describedby='theme-description'
      >
        {[
          { value: 'system', label: 'System', icon: IconThemeSystem },
          { value: 'light', label: 'Light', icon: IconThemeLight },
          { value: 'dark', label: 'Dark', icon: IconThemeDark },
        ].map((item) => (
          <RadioGroupItem key={item.value} item={item} isTheme />
        ))}
      </Radio>
      <div id='theme-description' className='sr-only'>
        {t('Choose between system preference, light mode, or dark mode')}
      </div>
    </div>
  )
}

const PRESET_OPTIONS: ReadonlyArray<{ value: ThemePreset; label: string }> = [
  { value: 'default', label: 'Default' },
  { value: 'underground', label: 'Underground' },
  { value: 'rose-garden', label: 'Rose Garden' },
  { value: 'lake-view', label: 'Lake View' },
  { value: 'sunset-glow', label: 'Sunset Glow' },
  { value: 'forest-whisper', label: 'Forest Whisper' },
  { value: 'ocean-breeze', label: 'Ocean Breeze' },
  { value: 'lavender-dream', label: 'Lavender Dream' },
] as const

function ThemePresetSwatch(props: { preset: ThemePreset }) {
  return (
    <div
      className={cn(
        'ring-border relative rounded-[6px] ring-[1px]',
        'group-data-[state=checked]:ring-primary group-data-[state=checked]:shadow-2xl',
        'group-focus-visible:ring-2'
      )}
    >
      <CircleCheck
        className={cn(
          'fill-primary absolute top-0 right-0 z-10 size-5 stroke-white',
          'translate-x-1/2 -translate-y-1/2',
          'group-data-[state=unchecked]:hidden'
        )}
        aria-hidden='true'
      />
      <div
        data-theme-preset={props.preset}
        className='flex h-10 overflow-hidden rounded-[6px]'
        role='img'
        aria-hidden='true'
      >
        <span className='bg-background border-border/40 flex-1 border-r' />
        <span className='bg-primary flex-1' />
        <span className='bg-secondary flex-1' />
        <span className='bg-accent flex-1' />
      </div>
    </div>
  )
}

function ThemePresetConfig() {
  const { t } = useTranslation()
  const { defaultPreset, preset, setPreset } = useThemeCustomization()
  return (
    <div>
      <SectionTitle
        title={t('Theme preset')}
        showReset={preset !== defaultPreset}
        onReset={() => setPreset(defaultPreset)}
      />
      <Radio
        value={preset}
        onValueChange={(v) => setPreset(v as ThemePreset)}
        className='grid w-full max-w-md grid-cols-4 gap-3'
        aria-label={t('Select theme preset')}
      >
        {PRESET_OPTIONS.map((p) => (
          <Item
            key={p.value}
            value={p.value}
            className='group transition duration-200 ease-in outline-none'
            aria-label={`Select ${p.label.toLowerCase()} preset`}
          >
            <ThemePresetSwatch preset={p.value} />
            <div className='mt-1 truncate text-center text-xs'>
              {t(p.label)}
            </div>
          </Item>
        ))}
      </Radio>
    </div>
  )
}

const RADIUS_OPTIONS: ReadonlyArray<{
  value: ThemeRadius
  label: string
  rem: string
}> = [
  { value: 'default', label: 'Auto', rem: 'var(--radius)' },
  { value: '0', label: '0', rem: '0rem' },
  { value: '0.3', label: '0.3', rem: '0.3rem' },
  { value: '0.5', label: '0.5', rem: '0.5rem' },
  { value: '0.75', label: '0.75', rem: '0.75rem' },
  { value: '1', label: '1.0', rem: '1rem' },
] as const

function RadiusConfig() {
  const { t } = useTranslation()
  const { defaultRadius, radius, setRadius } = useThemeCustomization()
  return (
    <div>
      <SectionTitle
        title={t('Radius')}
        showReset={radius !== defaultRadius}
        onReset={() => setRadius(defaultRadius)}
      />
      <Radio
        value={radius}
        onValueChange={(v) => setRadius(v as ThemeRadius)}
        className='grid w-full max-w-md grid-cols-6 gap-2'
        aria-label={t('Select corner radius')}
      >
        {RADIUS_OPTIONS.map((r) => (
          <Item
            key={r.value}
            value={r.value}
            className='group transition duration-200 ease-in outline-none'
            aria-label={`Set radius to ${r.label}`}
          >
            <div
              className={cn(
                'ring-border relative flex aspect-square items-end justify-end p-1.5 ring-[1px]',
                'group-data-[state=checked]:ring-primary group-data-[state=checked]:shadow-2xl',
                'group-focus-visible:ring-2'
              )}
              style={{ borderRadius: r.rem }}
            >
              <CircleCheck
                className={cn(
                  'fill-primary absolute top-0 right-0 z-10 size-5 stroke-white',
                  'translate-x-1/2 -translate-y-1/2',
                  'group-data-[state=unchecked]:hidden'
                )}
                aria-hidden='true'
              />
              <span
                className='border-foreground/70 size-4 border-t-2 border-r-2'
                style={{
                  borderTopRightRadius: r.rem,
                }}
                aria-hidden='true'
              />
            </div>
            <div className='mt-1 text-center text-xs'>{r.label}</div>
          </Item>
        ))}
      </Radio>
    </div>
  )
}

function SidebarConfig() {
  const { t } = useTranslation()
  const { defaultVariant, variant, setVariant } = useLayout()
  return (
    <div className='max-md:hidden'>
      <SectionTitle
        title={t('Sidebar')}
        showReset={defaultVariant !== variant}
        onReset={() => setVariant(defaultVariant)}
      />
      <Radio
        value={variant}
        onValueChange={setVariant}
        className='grid w-full max-w-md grid-cols-3 gap-4'
        aria-label={t('Select sidebar style')}
        aria-describedby='sidebar-description'
      >
        {[
          { value: 'inset', label: 'Inset', icon: IconSidebarInset },
          { value: 'floating', label: 'Floating', icon: IconSidebarFloating },
          { value: 'sidebar', label: 'Sidebar', icon: IconSidebarSidebar },
        ].map((item) => (
          <RadioGroupItem key={item.value} item={item} />
        ))}
      </Radio>
      <div id='sidebar-description' className='sr-only'>
        {t('Choose between inset, floating, or standard sidebar layout')}
      </div>
    </div>
  )
}

function LayoutConfig() {
  const { t } = useTranslation()
  const { open, setOpen } = useSidebar()
  const { defaultCollapsible, collapsible, setCollapsible } = useLayout()

  const radioState = open ? 'default' : collapsible

  return (
    <div className='max-md:hidden'>
      <SectionTitle
        title={t('Layout')}
        showReset={radioState !== 'default'}
        onReset={() => {
          setOpen(true)
          setCollapsible(defaultCollapsible)
        }}
      />
      <Radio
        value={radioState}
        onValueChange={(v) => {
          if (v === 'default') {
            setOpen(true)
            return
          }
          setOpen(false)
          setCollapsible(v as Collapsible)
        }}
        className='grid w-full max-w-md grid-cols-3 gap-4'
        aria-label={t('Select layout style')}
        aria-describedby='layout-description'
      >
        {[
          { value: 'default', label: 'Default', icon: IconLayoutDefault },
          { value: 'icon', label: 'Compact', icon: IconLayoutCompact },
          { value: 'offcanvas', label: 'Full layout', icon: IconLayoutFull },
        ].map((item) => (
          <RadioGroupItem key={item.value} item={item} />
        ))}
      </Radio>
      <div id='layout-description' className='sr-only'>
        {t(
          'Choose between default expanded, compact icon-only, or full layout mode'
        )}
      </div>
    </div>
  )
}

function DirConfig() {
  const { t } = useTranslation()
  const { defaultDir, dir, setDir } = useDirection()
  return (
    <div>
      <SectionTitle
        title={t('Direction')}
        showReset={defaultDir !== dir}
        onReset={() => setDir(defaultDir)}
      />
      <Radio
        value={dir}
        onValueChange={setDir}
        className='grid w-full max-w-md grid-cols-3 gap-4'
        aria-label={t('Select site direction')}
        aria-describedby='direction-description'
      >
        {[
          {
            value: 'ltr',
            label: 'Left to Right',
            icon: (props: SVGProps<SVGSVGElement>) => (
              <IconDir dir='ltr' {...props} />
            ),
          },
          {
            value: 'rtl',
            label: 'Right to Left',
            icon: (props: SVGProps<SVGSVGElement>) => (
              <IconDir dir='rtl' {...props} />
            ),
          },
        ].map((item) => (
          <RadioGroupItem key={item.value} item={item} />
        ))}
      </Radio>
      <div id='direction-description' className='sr-only'>
        {t('Choose between left-to-right or right-to-left site direction')}
      </div>
    </div>
  )
}
