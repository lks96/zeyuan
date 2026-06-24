<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Edit3, Link, Plus, Save, UserMinus, Users, X } from '@lucide/vue'
import {
  assignShopToUser,
  createUser,
  disableUser,
  fetchPermissions,
  fetchRolePermissions,
  fetchShops,
  fetchUserShops,
  fetchUsers,
  hasPermission,
  removeShopFromUser,
  updateRolePermissions,
  updateUser,
  type Permission,
  type Shop,
  type User,
  type UserShop,
} from '@/services/api'

type UserForm = {
  id?: number
  username: string
  password: string
  displayName: string
  role: User['role']
  status: User['status']
}

const users = ref<User[]>([])
const shops = ref<Shop[]>([])
const permissions = ref<Permission[]>([])
const rolePermissionCodes = ref<string[]>([])
const selectedRole = ref<User['role']>('user')
const isLoading = ref(true)
const isSaving = ref(false)
const apiError = ref('')
const successMessage = ref('')

const isUserModalOpen = ref(false)
const isAssignModalOpen = ref(false)
const selectedUser = ref<User | null>(null)
const currentAssignments = ref<UserShop[]>([])
const selectedShopIds = ref<number[]>([])
const assignmentRoles = ref<Record<number, UserShop['shopRole']>>({})

const userForm = ref<UserForm>({
  username: '',
  password: '',
  displayName: '',
  role: 'user',
  status: 'active',
})

const canCreateUser = computed(() => hasPermission('users:create'))
const canUpdateUser = computed(() => hasPermission('users:update'))
const canDisableUser = computed(() => hasPermission('users:disable'))
const canAssignShops = computed(() => hasPermission('users:assign_shops'))
const canViewPermissions = computed(() => hasPermission('permissions:view'))
const canManagePermissions = computed(() => hasPermission('permissions:manage'))
const canOperateUser = computed(() => canUpdateUser.value || canDisableUser.value || canAssignShops.value)
const isEditingUser = computed(() => Boolean(userForm.value.id))

const groupedPermissions = computed(() => {
  return permissions.value.reduce<Record<string, Permission[]>>((groups, permission) => {
    groups[permission.category] = groups[permission.category] ?? []
    groups[permission.category].push(permission)
    return groups
  }, {})
})

onMounted(loadPageData)

async function loadPageData() {
  isLoading.value = true
  apiError.value = ''

  try {
    const requests: Promise<unknown>[] = [
      fetchUsers().then((data) => {
        users.value = data
      }),
      fetchShops().then((data) => {
        shops.value = data
      }),
    ]

    if (canViewPermissions.value) {
      requests.push(
        fetchPermissions().then((data) => {
          permissions.value = data
        }),
      )
      requests.push(loadRolePermissions())
    }

    await Promise.all(requests)
  } catch {
    apiError.value = '无法加载用户管理数据，请检查权限或后端服务状态'
  } finally {
    isLoading.value = false
  }
}

async function loadRolePermissions() {
  rolePermissionCodes.value = await fetchRolePermissions(selectedRole.value)
}

function openCreateUser() {
  userForm.value = {
    username: '',
    password: '',
    displayName: '',
    role: 'user',
    status: 'active',
  }
  isUserModalOpen.value = true
}

function openEditUser(user: User) {
  userForm.value = {
    id: user.id,
    username: user.username,
    password: '',
    displayName: user.displayName,
    role: user.role,
    status: user.status,
  }
  isUserModalOpen.value = true
}

function closeUserModal() {
  isUserModalOpen.value = false
}

async function saveUser() {
  const validationError = validateUserForm()
  if (validationError) {
    apiError.value = validationError
    successMessage.value = ''
    return
  }

  isSaving.value = true
  apiError.value = ''
  successMessage.value = ''

  try {
    if (userForm.value.id) {
      await updateUser(userForm.value.id, {
        displayName: userForm.value.displayName,
        password: userForm.value.password || undefined,
        role: userForm.value.role,
        status: userForm.value.status,
      })
      successMessage.value = '用户已更新'
    } else {
      await createUser({
        username: userForm.value.username,
        password: userForm.value.password,
        displayName: userForm.value.displayName,
        role: userForm.value.role,
        status: userForm.value.status,
      })
      successMessage.value = '用户已创建'
    }

    closeUserModal()
    users.value = await fetchUsers()
  } catch (error) {
    apiError.value = apiErrorMessage(error, '保存用户失败，请检查表单内容')
  } finally {
    isSaving.value = false
  }
}

function validateUserForm() {
  const username = userForm.value.username.trim()
  const displayName = userForm.value.displayName.trim()
  const password = userForm.value.password

  if (!username || !displayName) {
    return '登录名和显示名称不能为空。'
  }
  if (!isEditingUser.value && !password) {
    return '新增用户时必须填写密码。'
  }
  if (password && password.length < 6) {
    return '密码至少需要 6 位。'
  }
  return ''
}

function apiErrorMessage(error: unknown, fallback: string) {
  if (typeof error === 'object' && error !== null && 'response' in error) {
    const response = (error as { response?: { data?: { error?: string } } }).response
    if (response?.data?.error) return response.data.error
  }
  return fallback
}

async function disableSelectedUser(user: User) {
  if (!window.confirm(`确认停用用户 ${user.displayName}？`)) return

  apiError.value = ''
  successMessage.value = ''
  try {
    await disableUser(user.id)
    users.value = await fetchUsers()
    successMessage.value = '用户已停用'
  } catch {
    apiError.value = '停用用户失败'
  }
}

async function openAssignShops(user: User) {
  selectedUser.value = user
  apiError.value = ''
  successMessage.value = ''

  try {
    currentAssignments.value = await fetchUserShops(user.id)
    selectedShopIds.value = currentAssignments.value.map((assignment) => assignment.shopId)
    assignmentRoles.value = currentAssignments.value.reduce<Record<number, UserShop['shopRole']>>((roles, assignment) => {
      roles[assignment.shopId] = assignment.shopRole
      return roles
    }, {})

    for (const shop of shops.value) {
      assignmentRoles.value[shop.id] = assignmentRoles.value[shop.id] ?? 'viewer'
    }

    isAssignModalOpen.value = true
  } catch {
    apiError.value = '加载店铺分配失败'
  }
}

function closeAssignModal() {
  isAssignModalOpen.value = false
  selectedUser.value = null
}

function isShopSelected(shopId: number) {
  return selectedShopIds.value.includes(shopId)
}

function toggleShop(shopId: number, checked: boolean) {
  if (checked && !selectedShopIds.value.includes(shopId)) {
    selectedShopIds.value = [...selectedShopIds.value, shopId]
  }
  if (!checked) {
    selectedShopIds.value = selectedShopIds.value.filter((id) => id !== shopId)
  }
}

async function saveShopAssignments() {
  if (!selectedUser.value) return

  isSaving.value = true
  apiError.value = ''
  successMessage.value = ''

  try {
    const currentShopIds = currentAssignments.value.map((assignment) => assignment.shopId)
    const toAddOrUpdate = selectedShopIds.value
    const toRemove = currentShopIds.filter((shopId) => !selectedShopIds.value.includes(shopId))

    await Promise.all([
      ...toAddOrUpdate.map((shopId) =>
        assignShopToUser(selectedUser.value!.id, {
          shopId,
          shopRole: assignmentRoles.value[shopId] ?? 'viewer',
        }),
      ),
      ...toRemove.map((shopId) => removeShopFromUser(selectedUser.value!.id, shopId)),
    ])

    successMessage.value = '店铺分配已保存'
    closeAssignModal()
  } catch {
    apiError.value = '保存店铺分配失败'
  } finally {
    isSaving.value = false
  }
}

async function switchPermissionRole(role: User['role']) {
  selectedRole.value = role
  apiError.value = ''
  try {
    await loadRolePermissions()
  } catch {
    apiError.value = '加载角色权限失败'
  }
}

function togglePermission(permissionCode: string, checked: boolean) {
  if (checked && !rolePermissionCodes.value.includes(permissionCode)) {
    rolePermissionCodes.value = [...rolePermissionCodes.value, permissionCode]
  }
  if (!checked) {
    rolePermissionCodes.value = rolePermissionCodes.value.filter((code) => code !== permissionCode)
  }
}

async function saveRolePermissions() {
  isSaving.value = true
  apiError.value = ''
  successMessage.value = ''

  try {
    rolePermissionCodes.value = await updateRolePermissions(selectedRole.value, rolePermissionCodes.value)
    successMessage.value = '角色权限已保存'
  } catch {
    apiError.value = '保存角色权限失败'
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <section class="page-panel">
    <div class="section-heading">
      <div>
        <p class="section-label">用户管理</p>
        <h2>账号与角色</h2>
      </div>
      <va-button v-if="canCreateUser" @click="openCreateUser">
        <Plus :size="18" />
        新增用户
      </va-button>
    </div>

    <va-alert v-if="apiError" color="warning" dense>
      {{ apiError }}
    </va-alert>
    <va-alert v-if="successMessage" color="success" dense>
      {{ successMessage }}
    </va-alert>

    <div class="data-table" :aria-busy="isLoading">
      <div class="data-table-row data-table-head user-table-row">
        <span>用户</span>
        <span>登录名</span>
        <span>角色</span>
        <span>状态</span>
        <span v-if="canOperateUser">操作</span>
      </div>
      <div v-for="user in users" :key="user.id" class="data-table-row user-table-row">
        <span class="entity-cell">
          <Users :size="18" />
          {{ user.displayName }}
        </span>
        <span>{{ user.username }}</span>
        <span>{{ user.role === 'admin' ? '管理员' : '普通用户' }}</span>
        <span>
          <va-chip size="small" :color="user.status === 'active' ? 'success' : 'danger'">
            {{ user.status }}
          </va-chip>
        </span>
        <span v-if="canOperateUser" class="row-actions">
          <va-button v-if="canUpdateUser" preset="secondary" size="small" @click="openEditUser(user)">
            <Edit3 :size="15" />
            编辑
          </va-button>
          <va-button v-if="canAssignShops" preset="secondary" size="small" @click="openAssignShops(user)">
            <Link :size="15" />
            分配店铺
          </va-button>
          <va-button
            v-if="canDisableUser && user.status === 'active'"
            preset="secondary"
            color="danger"
            size="small"
            @click="disableSelectedUser(user)"
          >
            <UserMinus :size="15" />
            停用
          </va-button>
        </span>
      </div>
    </div>
  </section>

  <section v-if="canViewPermissions" class="page-panel">
    <div class="section-heading">
      <div>
        <p class="section-label">权限配置</p>
        <h2>角色权限</h2>
      </div>
      <div class="segmented-actions">
        <va-button
          :preset="selectedRole === 'user' ? undefined : 'secondary'"
          size="small"
          @click="switchPermissionRole('user')"
        >
          普通用户
        </va-button>
        <va-button
          :preset="selectedRole === 'admin' ? undefined : 'secondary'"
          size="small"
          @click="switchPermissionRole('admin')"
        >
          管理员
        </va-button>
      </div>
    </div>

    <div class="permission-grid">
      <div v-for="(group, category) in groupedPermissions" :key="category" class="permission-group">
        <h3>{{ category }}</h3>
        <label v-for="permission in group" :key="permission.code" class="checkbox-row">
          <input
            type="checkbox"
            :checked="rolePermissionCodes.includes(permission.code)"
            :disabled="!canManagePermissions"
            @change="togglePermission(permission.code, ($event.target as HTMLInputElement).checked)"
          />
          <span>
            <strong>{{ permission.code }}</strong>
            <small>{{ permission.name }}</small>
          </span>
        </label>
      </div>
    </div>

    <div v-if="canManagePermissions" class="panel-actions">
      <va-button :loading="isSaving" @click="saveRolePermissions">
        <Save :size="18" />
        保存权限
      </va-button>
    </div>
  </section>

  <div v-if="isUserModalOpen" class="modal-backdrop" @click.self="closeUserModal">
    <form class="modal-panel" @submit.prevent="saveUser">
      <div class="modal-header">
        <h2>{{ isEditingUser ? '编辑用户' : '新增用户' }}</h2>
        <button type="button" class="icon-only" aria-label="关闭用户弹窗" @click="closeUserModal">
          <X :size="18" />
        </button>
      </div>

      <div class="form-grid">
        <label class="field-control">
          <span>登录名</span>
          <input v-model="userForm.username" :readonly="isEditingUser" required />
        </label>
        <label class="field-control">
          <span>显示名称</span>
          <input v-model="userForm.displayName" required />
        </label>
        <label class="field-control">
          <span>密码{{ isEditingUser ? '（留空不修改）' : '' }}</span>
          <input v-model="userForm.password" type="password" minlength="6" :required="!isEditingUser" />
        </label>
        <label class="field-control">
          <span>角色</span>
          <select v-model="userForm.role">
            <option value="user">普通用户</option>
            <option value="admin">管理员</option>
          </select>
        </label>
        <label class="field-control">
          <span>状态</span>
          <select v-model="userForm.status">
            <option value="active">active</option>
            <option value="disabled">disabled</option>
          </select>
        </label>
      </div>

      <div class="modal-actions">
        <va-button preset="secondary" type="button" @click="closeUserModal">取消</va-button>
        <va-button type="submit" :loading="isSaving">
          <Save :size="18" />
          保存
        </va-button>
      </div>
    </form>
  </div>

  <div v-if="isAssignModalOpen && selectedUser" class="modal-backdrop" @click.self="closeAssignModal">
    <form class="modal-panel modal-panel-wide" @submit.prevent="saveShopAssignments">
      <div class="modal-header">
        <h2>分配店铺：{{ selectedUser.displayName }}</h2>
        <button type="button" class="icon-only" aria-label="关闭分配弹窗" @click="closeAssignModal">
          <X :size="18" />
        </button>
      </div>

      <div class="assignment-list">
        <label v-for="shop in shops" :key="shop.id" class="assignment-row">
          <input
            type="checkbox"
            :checked="isShopSelected(shop.id)"
            @change="toggleShop(shop.id, ($event.target as HTMLInputElement).checked)"
          />
          <span>
            <strong>{{ shop.shopName }}</strong>
            <small>{{ shop.externalCode || shop.platform }}</small>
          </span>
          <select v-model="assignmentRoles[shop.id]" :disabled="!isShopSelected(shop.id)">
            <option value="owner">owner</option>
            <option value="operator">operator</option>
            <option value="viewer">viewer</option>
          </select>
        </label>
      </div>

      <div class="modal-actions">
        <va-button preset="secondary" type="button" @click="closeAssignModal">取消</va-button>
        <va-button type="submit" :loading="isSaving">
          <Save :size="18" />
          保存分配
        </va-button>
      </div>
    </form>
  </div>
</template>
