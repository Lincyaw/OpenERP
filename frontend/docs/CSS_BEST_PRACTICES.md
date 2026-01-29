# CSS Best Practices Guide

This document defines CSS best practices for the ERP frontend. All developers (including AI assistants) MUST follow these guidelines.

---

## Table of Contents

1. [Spacing System](#1-spacing-system)
2. [Responsive Design](#2-responsive-design)
3. [Specificity Management](#3-specificity-management)
4. [Layout Patterns](#4-layout-patterns)
5. [Semi Design Component Overrides](#5-semi-design-component-overrides)
6. [Common CSS Utilities](#6-common-css-utilities)

---

## 1. Spacing System

### Rule: Always use spacing tokens instead of hardcoded pixel values

The design system provides a 4px-based spacing scale via CSS variables. Using hardcoded values leads to inconsistent spacing and makes theme changes difficult.

### Available Tokens

```css
/* Base Scale (4px increments) */
--spacing-1: 4px --spacing-2: 8px --spacing-3: 12px --spacing-4: 16px --spacing-5: 20px
  --spacing-6: 24px --spacing-8: 32px --spacing-10: 40px --spacing-12: 48px --spacing-16: 64px
  /* Component-Specific Tokens */ --form-item-margin: var(--spacing-6)
  --card-padding: var(--spacing-6) --table-cell-padding-y: var(--spacing-3)
  --page-padding: var(--spacing-6) --page-padding-mobile: var(--spacing-4);
```

### Incorrect Examples (from codebase)

```css
/* MainLayout.css - hardcoded padding */
.main-layout__content {
  padding: 16px 24px; /* BAD: hardcoded values */
}

/* Header.css - hardcoded gap */
.header__left {
  gap: 12px; /* BAD: hardcoded value */
}

/* Form.css - hardcoded margins */
.form-actions {
  margin-top: 24px; /* BAD: hardcoded value */
  padding-top: 16px; /* BAD: hardcoded value */
}

/* Sidebar.css - hardcoded padding */
.sidebar__logo {
  padding: 0 16px; /* BAD: hardcoded value */
  gap: 12px; /* BAD: hardcoded value */
}
```

### Correct Examples

```css
/* MainLayout.css - using tokens */
.main-layout__content {
  padding: var(--spacing-4) var(--spacing-6); /* GOOD: uses tokens */
}

/* Header.css - using tokens */
.header__left {
  gap: var(--spacing-3); /* GOOD: uses token */
}

/* Form.css - using tokens */
.form-actions {
  margin-top: var(--spacing-6); /* GOOD: uses token */
  padding-top: var(--spacing-4); /* GOOD: uses token */
}

/* Sidebar.css - using tokens */
.sidebar__logo {
  padding: 0 var(--spacing-4); /* GOOD: uses token */
  gap: var(--spacing-3); /* GOOD: uses token */
}
```

### Quick Reference Table

| Hardcoded Value | Token Replacement |
| --------------- | ----------------- |
| 4px             | var(--spacing-1)  |
| 8px             | var(--spacing-2)  |
| 12px            | var(--spacing-3)  |
| 16px            | var(--spacing-4)  |
| 20px            | var(--spacing-5)  |
| 24px            | var(--spacing-6)  |
| 32px            | var(--spacing-8)  |
| 40px            | var(--spacing-10) |
| 48px            | var(--spacing-12) |
| 64px            | var(--spacing-16) |

---

## 2. Responsive Design

### Rule: Use Mobile-First approach with min-width media queries

Mobile-first ensures base styles work on all devices, with enhancements added for larger screens. Using `max-width` queries (desktop-first) leads to bloated mobile CSS.

### Breakpoints

```css
/* Mobile: base styles (no media query needed) */
/* Tablet: @media (min-width: 768px) */
/* Desktop: @media (min-width: 1024px) */
/* Wide: @media (min-width: 1440px) */
```

### Incorrect Examples (from codebase)

```css
/* Dashboard.css - desktop-first approach */
.dashboard-metrics .metric-card-wrapper {
  flex: 0 0 calc(16.666% - 14px); /* Desktop default */
}

@media screen and (max-width: 1200px) {
  .dashboard-metrics .metric-card-wrapper {
    flex: 0 0 calc(33.333% - 12px); /* BAD: overriding for smaller */
  }
}

@media screen and (max-width: 768px) {
  .dashboard-metrics .metric-card-wrapper {
    flex: 0 0 calc(50% - 8px); /* BAD: more overrides */
  }
}

@media screen and (max-width: 480px) {
  .dashboard-metrics .metric-card-wrapper {
    flex: 1 1 100%; /* BAD: even more overrides */
  }
}

/* Form.css - ordering problem */
@media (max-width: 768px) {
  /* tablet rules */
}

@media (max-width: 992px) {
  /* BAD: larger breakpoint after smaller */
  /* desktop rules - this order is confusing */
}
```

### Correct Examples

```css
/* Dashboard.css - mobile-first approach */
.dashboard-metrics .metric-card-wrapper {
  flex: 1 1 100%; /* Mobile: full width */
}

@media (min-width: 480px) {
  .dashboard-metrics .metric-card-wrapper {
    flex: 0 0 calc(50% - var(--spacing-2)); /* 2 columns */
  }
}

@media (min-width: 768px) {
  .dashboard-metrics .metric-card-wrapper {
    flex: 0 0 calc(33.333% - var(--spacing-3)); /* 3 columns */
  }
}

@media (min-width: 1200px) {
  .dashboard-metrics .metric-card-wrapper {
    flex: 0 0 calc(16.666% - var(--spacing-4)); /* 6 columns */
  }
}

/* Form.css - correct ordering */
.form-row {
  grid-template-columns: 1fr; /* Mobile: single column */
}

@media (min-width: 768px) {
  .form-row {
    grid-template-columns: repeat(2, 1fr); /* Tablet: 2 columns */
  }
}

@media (min-width: 1024px) {
  .form-row {
    grid-template-columns: repeat(4, 1fr); /* Desktop: 4 columns */
  }
}
```

### Mobile-First Checklist

- [ ] Base styles are mobile-optimized
- [ ] Media queries use `min-width` only
- [ ] Breakpoints are in ascending order
- [ ] No redundant style overrides

---

## 3. Specificity Management

### Rule: Avoid !important except for utility classes and accessibility

Using `!important` creates specificity wars and makes styles difficult to override. It should only be used for utility classes and accessibility requirements.

### Acceptable Uses of !important

```css
/* Utility classes */
.hidden {
  display: none !important;
}
.sr-only {
  position: absolute !important; /* ... */
}

/* Accessibility - reduced motion */
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}

/* Responsive visibility utilities */
.mobile-only {
  display: none !important;
}
@media (max-width: 767px) {
  .mobile-only {
    display: block !important;
  }
}
```

### Incorrect Examples (from codebase)

```css
/* SalesOrderForm.css - forcing styles on Semi components */
.section-title {
  margin-bottom: var(--spacing-4) !important; /* BAD: fighting Semi styles */
}

.section-header .section-title {
  margin-bottom: 0 !important; /* BAD: now fighting own styles */
}

.summary-section {
  padding: var(--page-padding-mobile) !important; /* BAD: unnecessary */
}

/* FormFieldWrapper.css - forcing border color */
.form-field-wrapper--error .semi-input {
  border-color: var(--semi-color-danger) !important; /* BAD: specificity issue */
}

/* TableToolbar.css - forcing gap */
.table-toolbar__actions {
  gap: var(--spacing-2) !important; /* BAD: should increase specificity instead */
}

/* Multiple pages - forcing width on Semi Select */
.filter-select .semi-select {
  width: 100% !important; /* BAD: common pattern, needs proper solution */
}
```

### Correct Examples

```css
/* SalesOrderForm.css - use proper specificity */
.sales-order-form .section-title {
  margin-bottom: var(--spacing-4); /* GOOD: scoped selector */
}

.sales-order-form .section-header .section-title {
  margin-bottom: 0; /* GOOD: more specific selector */
}

.sales-order-form .summary-section {
  padding: var(--page-padding-mobile); /* GOOD: no !important needed */
}

/* FormFieldWrapper.css - use data attribute for state */
.form-field-wrapper[data-error='true'] .semi-input,
.form-field-wrapper--error .semi-input-wrapper {
  border-color: var(--semi-color-danger); /* GOOD: proper targeting */
}

/* TableToolbar.css - use wrapper class */
.data-table .table-toolbar__actions {
  gap: var(--spacing-2); /* GOOD: increased specificity */
}

/* Filter select - use CSS variable override */
.filter-select {
  --semi-select-width: 100%; /* GOOD: use Semi's CSS variable */
}

/* Or use proper container scoping */
.page-filters .semi-select {
  width: 100%; /* GOOD: scoped to container */
}
```

### Strategies to Avoid !important

1. **Increase specificity with parent selector**

   ```css
   /* Instead of: */
   .component {
     margin: 0 !important;
   }

   /* Use: */
   .page .component {
     margin: 0;
   }
   ```

2. **Use data attributes for states**

   ```css
   /* Instead of: */
   .input--error {
     border-color: red !important;
   }

   /* Use: */
   .input[data-error='true'] {
     border-color: red;
   }
   ```

3. **Override CSS variables**

   ```css
   /* Instead of: */
   .semi-select {
     width: 100% !important;
   }

   /* Use: */
   .container {
     --semi-select-width: 100%;
   }
   ```

4. **Use :where() to reduce library specificity**
   ```css
   :where(.semi-button) {
     /* Lower specificity overrides */
   }
   ```

---

## 4. Layout Patterns

### Rule: Choose Flexbox or Grid based on layout requirements

- **Flexbox**: One-dimensional layouts (row OR column)
- **CSS Grid**: Two-dimensional layouts (row AND column)

### When to Use Flexbox

```css
/* Navigation bars */
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

/* Button groups */
.button-group {
  display: flex;
  gap: var(--spacing-2);
}

/* Vertical stacks */
.card-content {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
}

/* Centering */
.centered {
  display: flex;
  align-items: center;
  justify-content: center;
}
```

### When to Use CSS Grid

```css
/* Form layouts */
.form-row {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--spacing-4);
}

/* Dashboard cards */
.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: var(--spacing-4);
}

/* Complex page layouts */
.page-layout {
  display: grid;
  grid-template-columns: 1fr 300px;
  grid-template-rows: auto 1fr auto;
  gap: var(--spacing-6);
}

/* Table-like structures */
.data-grid {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 100px;
}
```

### Incorrect Examples (from codebase)

```css
/* Dashboard.css - using flexbox for 2D grid layout */
.dashboard-content {
  display: flex;
  flex-wrap: wrap; /* BAD: simulating grid with flexbox */
}

.dashboard-col-left {
  flex: 1 1 60%;
  min-width: 300px; /* BAD: magic numbers for responsive */
}

.dashboard-col-right {
  flex: 1 1 35%;
  min-width: 280px; /* BAD: inconsistent breakpoints */
}

/* Metric cards with calc() gymnastics */
.dashboard-metrics .metric-card-wrapper {
  flex: 0 0 calc(16.666% - 14px); /* BAD: complex calculations */
}
```

### Correct Examples

```css
/* Dashboard.css - using CSS Grid */
.dashboard-content {
  display: grid;
  grid-template-columns: 1fr; /* Mobile: single column */
  gap: var(--spacing-4);
}

@media (min-width: 1024px) {
  .dashboard-content {
    grid-template-columns: 2fr 1fr; /* Desktop: main + sidebar */
  }
}

/* Metric cards with auto-fit */
.dashboard-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--spacing-4);
}

/* No need for responsive overrides - auto-fit handles it! */
```

### Layout Component Usage

Use the provided layout components when possible:

```tsx
import { Grid, GridItem, Flex, Stack, Row } from '@/components/common/layout'

// CSS Grid layout
<Grid cols={{ mobile: 1, tablet: 2, desktop: 4 }} gap="md">
  <Card>Item 1</Card>
  <Card>Item 2</Card>
</Grid>

// Flexbox layouts
<Row justify="space-between" align="center">
  <Logo />
  <Navigation />
</Row>

<Stack gap="md">
  <FormField />
  <FormField />
</Stack>
```

---

## 5. Semi Design Component Overrides

### Rule: Use CSS variables and proper scoping to override Semi styles

Semi Design provides CSS variables for customization. Direct class overrides should use proper specificity.

### Available Semi CSS Variables

```css
:root {
  /* Colors */
  --semi-color-primary: #your-primary;
  --semi-color-primary-hover: #your-primary-hover;
  --semi-color-primary-active: #your-primary-active;

  /* Borders */
  --semi-border-radius-small: 2px;
  --semi-border-radius-medium: 4px;
  --semi-border-radius-large: 8px;

  /* Typography */
  --semi-font-size-small: 12px;
  --semi-font-size-regular: 14px;
  --semi-font-size-header-6: 16px;

  /* Spacing (use design tokens instead) */
}
```

### Incorrect Examples (from codebase)

```css
/* Direct Semi class override without scoping */
.semi-table-thead > tr > th {
  padding: var(--table-header-padding-y) var(--table-cell-padding-x);
  background-color: var(--semi-color-fill-0); /* BAD: affects all tables */
}

/* Using !important to fight Semi */
.filter-select .semi-select {
  width: 100% !important; /* BAD: !important war */
}

/* Magic numbers in Semi overrides */
.semi-navigation-item {
  min-height: 44px; /* BAD: hardcoded */
  padding-left: 12px; /* BAD: hardcoded */
}
```

### Correct Examples

```css
/* Scoped Semi class override */
.data-table .semi-table-thead > tr > th {
  padding: var(--table-header-padding-y) var(--table-cell-padding-x);
  background-color: var(--semi-color-fill-0); /* GOOD: scoped to data-table */
}

/* Using CSS variable for Semi component */
.page-filters {
  --semi-select-option-height: 40px;
}

.page-filters .semi-select-selection {
  width: 100%; /* GOOD: scoped, no !important */
}

/* Using design tokens in Semi overrides */
.sidebar__nav .semi-navigation-item {
  min-height: var(--spacing-11); /* GOOD: 44px via token */
  padding-left: var(--spacing-3); /* GOOD: 12px via token */
}
```

### Semi Override Strategies

1. **Use component wrapper class**

   ```css
   .custom-table .semi-table {
     /* styles */
   }
   ```

2. **Use Semi's CSS variables when available**

   ```css
   .container {
     --semi-color-primary: #your-color;
   }
   ```

3. **Use data attributes for custom states**

   ```css
   .semi-button[data-variant='custom'] {
     /* styles */
   }
   ```

4. **Override at the theme level for global changes**
   ```css
   /* In theme file, not component CSS */
   :root {
     --semi-border-radius-medium: var(--radius-md);
   }
   ```

---

## 6. Common CSS Utilities

### Rule: Use utility classes for common, single-purpose styles

The design system provides utility classes in `frontend/src/styles/utilities/common.css` for common patterns. Using utilities reduces duplication and improves consistency.

### Available Utility Files

| File                   | Purpose                                                     |
| ---------------------- | ----------------------------------------------------------- |
| `utilities/common.css` | General-purpose utilities (width, text, margin, flex, etc.) |
| `utilities/grid.css`   | Responsive grid column utilities                            |
| `utilities/form.css`   | Form-specific utilities (full-width selects, etc.)          |

### Width Utilities

```css
.w-full {
  width: 100%;
}
.w-auto {
  width: auto;
}
.max-w-sm {
  max-width: 24rem;
} /* 384px */
.max-w-md {
  max-width: 28rem;
} /* 448px */
.max-w-lg {
  max-width: 32rem;
} /* 512px */
.max-w-xl {
  max-width: 36rem;
} /* 576px */
```

### Semi Component Utilities

These utilities provide full-width styling for Semi components without `!important`:

```tsx
// Full-width select (from utilities/form.css)
<Select className="full-width-select" ... />

// Full-width input (from utilities/common.css)
<Input className="full-width-input" ... />

// Full-width datepicker (from utilities/form.css)
<DatePicker className="full-width-datepicker" ... />
```

### Text Utilities

```css
.text-truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.text-wrap {
  word-break: break-word;
}

.text-nowrap {
  white-space: nowrap;
}
```

### Margin/Padding Reset

```css
.m-0 {
  margin: 0;
}
.p-0 {
  padding: 0;
}
.mb-0 {
  margin-bottom: 0;
}
.mt-0 {
  margin-top: 0;
}
.ml-0 {
  margin-left: 0;
}
.mr-0 {
  margin-right: 0;
}
.pt-0 {
  padding-top: 0;
}
.pb-0 {
  padding-bottom: 0;
}
.pl-0 {
  padding-left: 0;
}
.pr-0 {
  padding-right: 0;
}
```

### Display Utilities

```css
.hidden {
  display: none !important;
} /* Allowed: utility class */
.invisible {
  visibility: hidden;
}
.visible {
  visibility: visible;
}
```

### Flex Utilities

```css
.flex {
  display: flex;
}
.flex-1 {
  flex: 1 1 0%;
}
.flex-auto {
  flex: 1 1 auto;
}
.flex-none {
  flex: none;
}
.flex-wrap {
  flex-wrap: wrap;
}
.flex-nowrap {
  flex-wrap: nowrap;
}
.flex-col {
  flex-direction: column;
}
.items-center {
  align-items: center;
}
.items-start {
  align-items: flex-start;
}
.items-end {
  align-items: flex-end;
}
.justify-center {
  justify-content: center;
}
.justify-between {
  justify-content: space-between;
}
.justify-end {
  justify-content: flex-end;
}
.justify-start {
  justify-content: flex-start;
}
```

### Gap Utilities (Using Design Tokens)

```css
.gap-1 {
  gap: var(--spacing-1);
} /* 4px */
.gap-2 {
  gap: var(--spacing-2);
} /* 8px */
.gap-3 {
  gap: var(--spacing-3);
} /* 12px */
.gap-4 {
  gap: var(--spacing-4);
} /* 16px */
.gap-6 {
  gap: var(--spacing-6);
} /* 24px */
.gap-8 {
  gap: var(--spacing-8);
} /* 32px */
```

### When to Use Utilities vs. Component CSS

**Use utilities when:**

- Applying a single, common style
- Quick layout adjustments
- Avoiding new CSS file creation for simple overrides

**Use component CSS when:**

- Multiple related styles form a cohesive component
- Complex responsive behavior needed
- Component-specific styling logic

### Examples

```tsx
// GOOD: Using utilities for simple layouts
<div className="flex items-center gap-2">
  <Icon />
  <span className="text-truncate">{longText}</span>
</div>

// GOOD: Using full-width utility for Semi component
<Select className="full-width-select" options={options} />

// BAD: Creating custom CSS for common pattern
// .my-wrapper { display: flex; align-items: center; gap: 8px; }

// BAD: Using !important for width override
// .my-select .semi-select { width: 100% !important; }
```

---

## Summary Checklist

Before submitting CSS changes, verify:

- [ ] All spacing values use `--spacing-*` tokens
- [ ] Responsive styles use mobile-first `min-width` queries
- [ ] No `!important` used (except utilities/accessibility)
- [ ] Layout uses appropriate pattern (Flexbox vs Grid)
- [ ] Semi overrides are properly scoped
- [ ] No hardcoded pixel values for margins, padding, or gaps
- [ ] Media query breakpoints are in ascending order
- [ ] Common patterns use utility classes when appropriate

---

## Related Documents

- [Frontend README](../README.md) - Design System overview
- [Spacing Tokens](../src/styles/tokens/spacing.css) - Available spacing values
- [Breakpoints](../src/styles/tokens/breakpoints.css) - Responsive breakpoints
- [Common Utilities](../src/styles/utilities/common.css) - Utility classes
- [Form Utilities](../src/styles/utilities/form.css) - Form-specific utilities
- [Grid Utilities](../src/styles/utilities/grid.css) - Responsive grid utilities
