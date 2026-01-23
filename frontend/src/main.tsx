import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { getRouteObjects } from './router'

// Design system styles (tokens, accessibility, utilities)
import './styles/index.css'

// Base application styles
import './index.css'

// Create browser router with route configuration
const router = createBrowserRouter(getRouteObjects())

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>
)
