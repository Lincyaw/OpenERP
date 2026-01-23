# ERP Frontend Design System

This document defines the design system and frontend development guidelines for the ERP application.

> **IMPORTANT**: All frontend developers (including AI assistants) MUST follow these guidelines when implementing UI components and pages.

## Table of Contents

1. [Technology Stack](#technology-stack)
2. [Responsive Design](#responsive-design)
3. [CSS Architecture](#css-architecture)
4. [Design Tokens](#design-tokens)
5. [Layout Components](#layout-components)
6. [Theme System](#theme-system)
7. [Accessibility (WCAG 2.1 AA)](#accessibility-wcag-21-aa)
8. [Interaction Design](#interaction-design)
9. [Component Guidelines](#component-guidelines)
10. [Best Practices](#best-practices)

---

## Technology Stack

| Category | Technology | Purpose |
|----------|------------|---------|
| Framework | React 19+ | UI framework |
| Language | TypeScript | Type safety |
| UI Library | Semi Design | Component library |
| State | Zustand | Global state management |
| Forms | React Hook Form + Zod | Form handling & validation |
| Routing | React Router v7 | Navigation |
| Build | Vite | Development & bundling |

---

## Responsive Design

### Breakpoints

We use a **mobile-first** approach with 4 breakpoints:

| Breakpoint | Width | CSS Variable | Usage |
|------------|-------|--------------|-------|
| Mobile | 375px | `--breakpoint-mobile` | Default (base) |
| Tablet | 768px | `--breakpoint-tablet` | `@media (min-width: 768px)` |
| Desktop | 1024px | `--breakpoint-desktop` | `@media (min-width: 1024px)` |
| Wide | 1440px | `--breakpoint-wide` | `@media (min-width: 1440px)` |

### Usage Example

```css
/* Mobile first - styles apply to all sizes */
.card {
  padding: var(--spacing-4);
}

/* Tablet and up */
@media (min-width: 768px) {
  .card {
    padding: var(--spacing-6);
  }
}

/* Desktop and up */
@media (min-width: 1024px) {
  .card {
    padding: var(--spacing-8);
  }
}
```

### Container Max-Widths

| Breakpoint | Max-Width |
|------------|-----------|
| Mobile | 100% |
| Tablet | 720px |
| Desktop | 960px |
| Wide | 1320px |

---

## CSS Architecture

### Structure

```
src/styles/
├── tokens/
│   ├── index.css        # Main token imports
│   ├── breakpoints.css  # Responsive breakpoints
│   ├── colors.css       # Color palette
│   ├── typography.css   # Font styles
│   ├── spacing.css      # Spacing scale
│   ├── shadows.css      # Shadow elevation
│   └── animations.css   # Motion tokens
├── utilities/
│   └── grid.css         # Grid utilities
├── accessibility.css    # A11y utilities
└── index.css           # Main entry point
```

### Conventions

1. **CSS Variables for tokens**: Always use design tokens via CSS variables
2. **CSS Modules for components**: Component-specific styles use CSS Modules
3. **BEM naming for global classes**: Block__Element--Modifier pattern

```css
/* BEM naming example */
.card { }
.card__header { }
.card__header--highlighted { }
.card__body { }
.card__footer { }
```

---

## Design Tokens

### Colors

```css
/* Primary (Brand) */
--color-primary          /* Main brand color */
--color-primary-hover    /* Hover state */
--color-primary-active   /* Active/pressed state */
--color-primary-light    /* Light variant for backgrounds */

/* Semantic */
--color-success          /* Positive actions/status */
--color-warning          /* Caution/attention */
--color-danger           /* Error/destructive */

/* Neutral (Gray Scale) */
--color-neutral-50 to --color-neutral-900

/* Backgrounds */
--color-bg-layout        /* Page background */
--color-bg-container     /* Card/panel background */
--color-bg-elevated      /* Elevated surfaces */

/* Text */
--color-text-primary     /* Main content */
--color-text-secondary   /* Secondary content */
--color-text-tertiary    /* Muted content */
--color-text-disabled    /* Disabled state */

/* Borders */
--color-border-primary   /* Default borders */
--color-border-secondary /* Subtle borders */
```

### Spacing

Based on 4px grid system:

```css
--spacing-1   /* 4px */
--spacing-2   /* 8px */
--spacing-3   /* 12px */
--spacing-4   /* 16px */
--spacing-5   /* 20px */
--spacing-6   /* 24px */
--spacing-8   /* 32px */
--spacing-10  /* 40px */
--spacing-12  /* 48px */
--spacing-16  /* 64px */
```

### Typography

```css
/* Font sizes (rem-based for scaling) */
--font-size-xs    /* 0.75rem / 12px */
--font-size-sm    /* 0.875rem / 14px */
--font-size-base  /* 1rem / 16px */
--font-size-lg    /* 1.125rem / 18px */
--font-size-xl    /* 1.25rem / 20px */
--font-size-2xl   /* 1.5rem / 24px */
--font-size-3xl   /* 1.875rem / 30px */
--font-size-4xl   /* 2.25rem / 36px */

/* Font weights */
--font-weight-normal    /* 400 */
--font-weight-medium    /* 500 */
--font-weight-semibold  /* 600 */
--font-weight-bold      /* 700 */

/* Line heights */
--line-height-tight     /* 1.25 */
--line-height-normal    /* 1.5 */
--line-height-relaxed   /* 1.625 */
```

### Shadows

```css
--shadow-sm   /* Subtle (cards) */
--shadow-md   /* Default (raised elements) */
--shadow-lg   /* Pronounced (popovers) */
--shadow-xl   /* Strong (modals) */
--shadow-2xl  /* Maximum (overlay dialogs) */
```

### Animations

```css
/* Durations */
--duration-fast     /* 150ms */
--duration-normal   /* 200ms */
--duration-slow     /* 300ms */

/* Easing */
--ease-default      /* Standard motion */
--ease-in           /* Enter animations */
--ease-out          /* Exit animations */
--ease-bounce       /* Playful interactions */
```

---

## Layout Components

### Container

Centers content with responsive max-width:

```tsx
import { Container } from '@/components/common'

<Container size="lg" padding="md">
  <PageContent />
</Container>
```

### Grid

CSS Grid-based responsive layouts:

```tsx
import { Grid, GridItem } from '@/components/common'

// Responsive grid
<Grid cols={{ mobile: 1, tablet: 2, desktop: 4 }} gap="md">
  <Card>Item 1</Card>
  <Card>Item 2</Card>
  <Card>Item 3</Card>
  <Card>Item 4</Card>
</Grid>

// 12-column layout
<Grid cols={12} gap="lg">
  <GridItem span={8}>Main Content</GridItem>
  <GridItem span={4}>Sidebar</GridItem>
</Grid>
```

### Flex / Stack / Row

Flexbox utilities:

```tsx
import { Flex, Stack, Row, Spacer, Divider } from '@/components/common'

// Horizontal row with space between
<Row justify="space-between" align="center">
  <Logo />
  <Navigation />
</Row>

// Vertical stack
<Stack gap="md">
  <FormField />
  <FormField />
  <Button>Submit</Button>
</Stack>

// Spacer pushes items apart
<Row>
  <Logo />
  <Spacer />
  <Button>Login</Button>
</Row>
```

---

## Theme System

### Available Themes

| Theme | Description |
|-------|-------------|
| `light` | Default light theme |
| `dark` | Dark mode for low-light environments |
| `elder` | High contrast, larger text for accessibility |

### Theme Usage

```tsx
import { useThemeManager } from '@/hooks'

function ThemeToggle() {
  const { theme, setTheme, toggleTheme } = useThemeManager()

  return (
    <Select value={theme} onChange={setTheme}>
      <Option value="light">浅色</Option>
      <Option value="dark">深色</Option>
      <Option value="elder">适老化</Option>
    </Select>
  )
}
```

### Font Scaling

For accessibility, support user font size preferences:

```tsx
import { useFontScale } from '@/hooks'

function FontScaleSelector() {
  const { fontScale, setFontScale } = useFontScale()

  return (
    <Select value={fontScale} onChange={setFontScale}>
      <Option value="default">默认</Option>
      <Option value="medium">中等 (112.5%)</Option>
      <Option value="large">大 (125%)</Option>
      <Option value="xlarge">超大 (137.5%)</Option>
    </Select>
  )
}
```

---

## Accessibility (WCAG 2.1 AA)

### Requirements

1. **Color Contrast**: Minimum 4.5:1 for normal text, 3:1 for large text
2. **Focus Indicators**: All interactive elements must have visible focus
3. **Touch Targets**: Minimum 44x44px tap targets
4. **Screen Reader Support**: Proper ARIA labels and roles
5. **Keyboard Navigation**: All features accessible via keyboard
6. **Reduced Motion**: Respect `prefers-reduced-motion`

### Focus Management

```css
/* Default focus style */
:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
```

### Screen Reader Utilities

```tsx
// Visually hidden but accessible
<span className="sr-only">Description for screen readers</span>

// Skip link for keyboard users
<a href="#main-content" className="skip-link">
  跳到主要内容
</a>
```

### Accessibility Hooks

```tsx
import { useAccessibilityPreferences } from '@/hooks'

function Component() {
  const { reducedMotion, highContrast } = useAccessibilityPreferences()

  return (
    <div className={reducedMotion ? 'no-animation' : ''}>
      Content
    </div>
  )
}
```

---

## Interaction Design

### Principles

1. **最少点击原则 (Minimum Clicks)**: Reduce steps to complete tasks
2. **智能默认值 (Smart Defaults)**: Pre-fill with sensible defaults
3. **批量操作 (Batch Operations)**: Support multi-select actions
4. **快捷键 (Keyboard Shortcuts)**: Provide shortcuts for power users

### Form UX Guidelines

- Auto-focus first input on page load
- Show validation errors inline, not in alerts
- Preserve form data on navigation
- Disable submit button when form is invalid or submitting

### Table UX Guidelines

- Support column sorting (click header)
- Support column filtering
- Show loading skeleton, not spinner
- Provide bulk actions for selected rows

### Loading States

- Use skeleton screens instead of spinners where possible
- Show progress indicators for long operations
- Disable interactive elements during loading

---

## Component Guidelines

### Semi Design Integration

Use Semi Design components as the base:

```tsx
import { Button, Table, Form, Input } from '@douyinfe/semi-ui'
import { IconPlus, IconSearch } from '@douyinfe/semi-icons'
```

### Custom Components Location

```
src/components/
├── common/          # Reusable components
│   ├── form/        # Form fields
│   └── layout/      # Layout utilities
├── layout/          # App layout (Header, Sidebar)
└── [feature]/       # Feature-specific components
```

### Component Props Convention

```tsx
interface ComponentProps {
  /** Required props first */
  id: string

  /** Optional props with defaults */
  variant?: 'primary' | 'secondary'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean

  /** Event handlers */
  onChange?: (value: string) => void
  onClick?: () => void

  /** Styling props last */
  className?: string
  style?: CSSProperties
}
```

---

## Best Practices

### DO

✅ Use design tokens for all colors, spacing, and typography
✅ Test on all breakpoints (mobile, tablet, desktop)
✅ Provide loading and error states
✅ Support keyboard navigation
✅ Use semantic HTML elements
✅ Add ARIA labels for icons and interactive elements
✅ Test with screen reader
✅ Respect user's motion preferences

### DON'T

❌ Hardcode colors, sizes, or spacing values
❌ Use `!important` except for utilities
❌ Create components with fixed widths
❌ Rely solely on color to convey information
❌ Remove focus outlines without providing alternatives
❌ Use auto-playing media without user consent
❌ Block zoom or text scaling

---

## File Structure

```
frontend/
├── src/
│   ├── api/              # Auto-generated API client (DO NOT EDIT)
│   ├── assets/           # Static assets
│   ├── components/
│   │   ├── common/       # Reusable components
│   │   │   ├── form/     # Form fields
│   │   │   └── layout/   # Layout utilities
│   │   └── layout/       # App layout
│   ├── features/         # Feature modules
│   ├── hooks/            # Custom hooks
│   ├── pages/            # Route pages
│   ├── router/           # Routing config
│   ├── services/         # API services
│   ├── store/            # Zustand stores
│   ├── styles/           # Design system styles
│   │   ├── tokens/       # Design tokens
│   │   └── utilities/    # CSS utilities
│   ├── types/            # TypeScript types
│   └── utils/            # Utility functions
└── README.md             # This file
```

---

## Scripts

```bash
# Development
npm run dev           # Start dev server

# Build
npm run build         # Production build
npm run preview       # Preview build

# Quality
npm run lint          # Run ESLint
npm run lint:fix      # Fix ESLint issues
npm run format        # Format with Prettier
npm run type-check    # TypeScript check

# API
npm run api:generate  # Generate API client from OpenAPI spec
```
