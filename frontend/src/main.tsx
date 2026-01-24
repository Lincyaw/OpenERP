import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { getRouteObjects } from './router'
import { AuthProvider } from './components/auth'

// Design system styles (tokens, accessibility, utilities)
import './styles/index.css'

// Base application styles
import './index.css'

// Create browser router with route configuration
const router = createBrowserRouter(getRouteObjects())

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AuthProvider>
      <RouterProvider router={router} />
    </AuthProvider>
  </StrictMode>
)
