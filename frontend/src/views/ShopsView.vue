<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Edit3, ExternalLink, Plus, Save, Store, X, XCircle } from '@lucide/vue'
import {
  closeShop,
  createShop,
  fetchShops,
  hasPermission,
  updateShop,
  type CreateShopPayload,
  type Shop,
  type UpdateShopPayload,
} from '@/services/api'

type ShopForm = {
  id?: number
  shopName: string
  platform: string
  externalCode: string
  euRepresentative: string
  status: Shop['status']
}

const shops = ref<Shop[]>([])
const isLoading = ref(true)
const isSaving = ref(false)
const apiError = ref('')
const successMessage = ref('')
const isShopModalOpen = ref(false)

const emptyShopForm = (): ShopForm => ({
  shopName: '',
  platform: 'temu',
  externalCode: '',
  euRepresentative: '',
  status: 'active',
})

const shopForm = ref<ShopForm>(emptyShopForm())

const canCreateShop = computed(() => hasPermission('shops:create'))
const canUpdateShop = computed(() => hasPermission('shops:update'))
const canDeleteShop = computed(() => hasPermission('shops:delete'))
const canOperateShop = computed(() => canUpdateShop.value || canDeleteShop.value)
const isEditingShop = computed(() => Boolean(shopForm.value.id))

const statusLabels: Record<Shop['status'], string> = {
  active: '启用',
  paused: '暂停',
  closed: '已关闭',
}

const roleLabels: Record<NonNullable<Shop['shopRole']>, string> = {
  owner: '店铺负责人',
  operator: '运营',
  viewer: '只读',
}

onMounted(loadShops)

async function loadShops() {
  isLoading.value = true
  apiError.value = ''

  try {
    shops.value = await fetchShops()
  } catch {
    apiError.value = '无法加载店铺数据，请检查登录状态、权限或后端服务。'
  } finally {
    isLoading.value = false
  }
}

function openCreateShop() {
  shopForm.value = emptyShopForm()
  apiError.value = ''
  successMessage.value = ''
  isShopModalOpen.value = true
}

function openEditShop(shop: Shop) {
  shopForm.value = {
    id: shop.id,
    shopName: shop.shopName,
    platform: shop.platform,
    externalCode: shop.externalCode,
    euRepresentative: shop.euRepresentative,
    status: shop.status,
  }
  apiError.value = ''
  successMessage.value = ''
  isShopModalOpen.value = true
}

function closeShopModal() {
  isShopModalOpen.value = false
}

async function saveShop() {
  isSaving.value = true
  apiError.value = ''
  successMessage.value = ''

  try {
    if (shopForm.value.id) {
      const payload: UpdateShopPayload = {
        shopName: shopForm.value.shopName,
        platform: shopForm.value.platform,
        externalCode: shopForm.value.externalCode,
        euRepresentative: shopForm.value.euRepresentative,
        status: shopForm.value.status,
      }
      await updateShop(shopForm.value.id, payload)
      successMessage.value = '店铺已更新。'
    } else {
      const payload: CreateShopPayload = {
        shopName: shopForm.value.shopName,
        platform: shopForm.value.platform,
        externalCode: shopForm.value.externalCode,
        euRepresentative: shopForm.value.euRepresentative,
        status: shopForm.value.status,
      }
      await createShop(payload)
      successMessage.value = '店铺已创建。'
    }

    closeShopModal()
    await loadShops()
  } catch {
    apiError.value = '保存店铺失败，请检查店铺名称、平台和店铺编码是否正确。'
  } finally {
    isSaving.value = false
  }
}

async function closeSelectedShop(shop: Shop) {
  if (!window.confirm(`确认关闭店铺“${shop.shopName}”？关闭后仍会保留历史数据。`)) return

  isSaving.value = true
  apiError.value = ''
  successMessage.value = ''

  try {
    await closeShop(shop.id)
    await loadShops()
    successMessage.value = '店铺已关闭。'
  } catch {
    apiError.value = '关闭店铺失败，请稍后重试。'
  } finally {
    isSaving.value = false
  }
}

function shopRoleLabel(shop: Shop) {
  if (!shop.shopRole) return '管理员'
  return roleLabels[shop.shopRole]
}

function statusColor(status: Shop['status']) {
  if (status === 'active') return 'success'
  if (status === 'paused') return 'warning'
  return 'danger'
}
</script>

<template>
  <section class="page-panel">
    <div class="section-heading">
      <div>
        <p class="section-label">店铺管理</p>
        <h2>店铺访问范围</h2>
      </div>
      <va-button v-if="canCreateShop" @click="openCreateShop">
        <Plus :size="18" />
        新增店铺
      </va-button>
    </div>

    <va-alert v-if="apiError" color="warning" dense>
      {{ apiError }}
    </va-alert>
    <va-alert v-if="successMessage" color="success" dense>
      {{ successMessage }}
    </va-alert>

    <div class="data-table" :aria-busy="isLoading">
      <div class="data-table-row data-table-head shop-table-row">
        <span>店铺</span>
        <span>平台</span>
        <span>店铺编码</span>
        <span>欧代</span>
        <span>店铺URL</span>
        <span>权限</span>
        <span>状态</span>
        <span v-if="canOperateShop">操作</span>
      </div>

      <div v-if="!isLoading && shops.length === 0" class="empty-state">
        暂无店铺数据
      </div>

      <div v-for="shop in shops" :key="shop.id" class="data-table-row shop-table-row">
        <span class="entity-cell">
          <Store :size="18" />
          {{ shop.shopName }}
        </span>
        <span>{{ shop.platform }}</span>
        <span>{{ shop.externalCode || '-' }}</span>
        <span>{{ shop.euRepresentative || '-' }}</span>
        <span>
          <a v-if="shop.shopUrl" :href="shop.shopUrl" target="_blank" rel="noreferrer" class="text-link">
            打开
            <ExternalLink :size="14" />
          </a>
          <template v-else>-</template>
        </span>
        <span>{{ shopRoleLabel(shop) }}</span>
        <span>
          <va-chip size="small" :color="statusColor(shop.status)">
            {{ statusLabels[shop.status] }}
          </va-chip>
        </span>
        <span v-if="canOperateShop" class="row-actions">
          <va-button v-if="canUpdateShop" preset="secondary" size="small" @click="openEditShop(shop)">
            <Edit3 :size="15" />
            编辑
          </va-button>
          <va-button
            v-if="canDeleteShop && shop.status !== 'closed'"
            preset="secondary"
            color="danger"
            size="small"
            :disabled="isSaving"
            @click="closeSelectedShop(shop)"
          >
            <XCircle :size="15" />
            关闭
          </va-button>
        </span>
      </div>
    </div>
  </section>

  <div v-if="isShopModalOpen" class="modal-backdrop" @click.self="closeShopModal">
    <form class="modal-panel" @submit.prevent="saveShop">
      <div class="modal-header">
        <h2>{{ isEditingShop ? '编辑店铺' : '新增店铺' }}</h2>
        <button type="button" class="icon-only" aria-label="关闭店铺弹窗" @click="closeShopModal">
          <X :size="18" />
        </button>
      </div>

      <div class="form-grid">
        <label class="field-control">
          <span>店铺名称</span>
          <input v-model.trim="shopForm.shopName" required />
        </label>
        <label class="field-control">
          <span>平台</span>
          <input v-model.trim="shopForm.platform" required />
        </label>
        <label class="field-control">
          <span>店铺编码</span>
          <input v-model.trim="shopForm.externalCode" placeholder="例如 supplierId" />
        </label>
        <label class="field-control">
          <span>欧代</span>
          <input v-model.trim="shopForm.euRepresentative" placeholder="可选" />
        </label>
        <label class="field-control">
          <span>状态</span>
          <select v-model="shopForm.status">
            <option value="active">启用</option>
            <option value="paused">暂停</option>
            <option value="closed">已关闭</option>
          </select>
        </label>
      </div>

      <div class="modal-actions">
        <va-button preset="secondary" type="button" @click="closeShopModal">取消</va-button>
        <va-button type="submit" :loading="isSaving">
          <Save :size="18" />
          保存
        </va-button>
      </div>
    </form>
  </div>
</template>
