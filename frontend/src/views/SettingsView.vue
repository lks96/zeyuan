<script setup lang="ts">
import { computed, ref } from 'vue'
import { Save } from '@lucide/vue'
import { hasPermission } from '@/services/api'

const syncOptions = ['15 分钟', '30 分钟', '1 小时', '手动同步']
const syncInterval = ref('30 分钟')
const canUpdateSettings = computed(() => hasPermission('settings:update'))
</script>

<template>
  <section class="page-panel settings-panel">
    <div class="section-heading">
      <div>
        <p class="section-label">系统设置</p>
        <h2>基础配置</h2>
      </div>
      <va-button v-if="canUpdateSettings">
        <Save :size="18" />
        保存
      </va-button>
      <va-chip v-else color="secondary">只读</va-chip>
    </div>

    <div class="settings-grid">
      <va-input label="API 地址" model-value="http://localhost:8080" :readonly="!canUpdateSettings" />
      <va-input label="店铺别名" placeholder="例如：主店铺" :readonly="!canUpdateSettings" />
      <label class="field-control">
        <span>同步间隔</span>
        <select v-model="syncInterval" :disabled="!canUpdateSettings">
          <option v-for="option in syncOptions" :key="option" :value="option">
            {{ option }}
          </option>
        </select>
      </label>
      <va-input label="Webhook 地址" placeholder="https://example.com/webhook" :readonly="!canUpdateSettings" />
    </div>
  </section>
</template>
