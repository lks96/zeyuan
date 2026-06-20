import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '@/views/DashboardView.vue'
import LoginView from '@/views/LoginView.vue'
import SettingsView from '@/views/SettingsView.vue'
import ShopsView from '@/views/ShopsView.vue'
import ToolsView from '@/views/ToolsView.vue'
import UsersView from '@/views/UsersView.vue'
import { clearStoredSession, getStoredPermissions, getStoredUser, hasPermission } from '@/services/api'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginView,
    },
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView,
      meta: { permission: 'dashboard:view' },
    },
    {
      path: '/tools',
      name: 'tools',
      component: ToolsView,
      meta: { permission: 'tools:view' },
    },
    {
      path: '/shops',
      name: 'shops',
      component: ShopsView,
      meta: { permission: 'shops:view' },
    },
    {
      path: '/users',
      name: 'users',
      component: UsersView,
      meta: { permission: 'users:view' },
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsView,
      meta: { permission: 'settings:view' },
    },
  ],
})

router.beforeEach((to) => {
  const isLoggedIn = Boolean(getStoredUser())
  const permission = typeof to.meta.permission === 'string' ? to.meta.permission : undefined
  if (to.name !== 'login' && !isLoggedIn) {
    return {
      name: 'login',
      query: {
        redirect: to.fullPath,
      },
    }
  }

  if (to.name !== 'login' && getStoredPermissions().length === 0) {
    clearStoredSession()
    return {
      name: 'login',
      query: {
        redirect: to.fullPath,
      },
    }
  }

  if (to.name !== 'login' && !hasPermission(permission)) {
    return { name: 'dashboard' }
  }

  if (to.name === 'login' && isLoggedIn) {
    const redirect = to.query.redirect
    if (typeof redirect === 'string' && redirect.startsWith('/')) {
      return redirect
    }
    return { name: 'dashboard' }
  }

  return true
})

export default router
