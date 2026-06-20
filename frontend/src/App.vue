<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { Bell, Blocks, LayoutDashboard, LogOut, Plus, Settings, Store, Users } from '@lucide/vue'
import { clearStoredSession, fetchCurrentSession, hasPermission } from '@/services/api'

const route = useRoute()
const router = useRouter()
const permissionVersion = ref(0)

const navItems = [
  { label: '仪表盘', icon: LayoutDashboard, to: '/', permission: 'dashboard:view' },
  { label: '工具中心', icon: Blocks, to: '/tools', permission: 'tools:view' },
  { label: '店铺管理', icon: Store, to: '/shops', permission: 'shops:view' },
  { label: '用户管理', icon: Users, to: '/users', permission: 'users:view' },
  { label: '系统设置', icon: Settings, to: '/settings', permission: 'settings:view' },
]

const visibleNavItems = computed(() => {
  permissionVersion.value
  return navItems.filter((item) => hasPermission(item.permission))
})
const activeTitle = computed(() => navItems.find((item) => item.to === route.path)?.label ?? 'Temu Tools')
const isLoginRoute = computed(() => route.name === 'login')
const canCreateTask = computed(() => {
  permissionVersion.value
  return hasPermission('tasks:create')
})

async function syncCurrentSession() {
  if (isLoginRoute.value) return

  try {
    await fetchCurrentSession()
    permissionVersion.value += 1
  } catch {
    clearStoredSession()
    router.push('/login')
  }
}

onMounted(syncCurrentSession)

watch(
  () => route.name,
  () => {
    syncCurrentSession()
  },
)

function logout() {
  clearStoredSession()
  permissionVersion.value += 1
  router.push('/login')
}
</script>

<template>
  <RouterView v-if="isLoginRoute" />

  <div v-else class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">T</div>
        <div>
          <div class="brand-name">Temu Tools</div>
          <div class="brand-subtitle">运营工具台</div>
        </div>
      </div>

      <nav class="nav-list">
        <button
          v-for="item in visibleNavItems"
          :key="item.to"
          class="nav-item"
          :class="{ active: route.path === item.to }"
          type="button"
          @click="router.push(item.to)"
        >
          <component :is="item.icon" class="nav-icon" :size="18" />
          <span>{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <div class="workspace">
      <header class="topbar">
        <div>
          <p class="eyebrow">Temu Tools</p>
          <h1>{{ activeTitle }}</h1>
        </div>
        <div class="topbar-actions">
          <va-button preset="secondary" class="icon-button" aria-label="通知">
            <Bell :size="18" />
          </va-button>
          <va-button v-if="canCreateTask">
            <Plus :size="18" />
            新建任务
          </va-button>
          <va-button preset="secondary" @click="logout">
            <LogOut :size="18" />
            退出
          </va-button>
        </div>
      </header>

      <main class="content">
        <RouterView />
      </main>
    </div>
  </div>
</template>
