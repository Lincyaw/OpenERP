import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { getRouteObjects } from './router'
import { AuthProvider } from './components/auth'
import { I18nProvider } from './components/providers'

// Initialize i18n (must be imported before components that use translations)
import './i18n'

// Design system styles (tokens, accessibility, utilities)
import './styles/index.css'

// Base application styles
import './index.css'

// Create browser router with route configuration
const router = createBrowserRouter(getRouteObjects())

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <I18nProvider>
      <AuthProvider>
        <RouterProvider router={router} />
      </AuthProvider>
    </I18nProvider>
  </StrictMode>
)
