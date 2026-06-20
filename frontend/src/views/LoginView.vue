<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogIn } from '@lucide/vue'
import { login } from '@/services/api'

const route = useRoute()
const router = useRouter()

const username = ref('')
const password = ref('')
const isSubmitting = ref(false)
const errorMessage = ref('')

const redirectPath = computed(() => {
  const redirect = route.query.redirect
  return typeof redirect === 'string' && redirect.startsWith('/') ? redirect : '/'
})

async function submitLogin() {
  errorMessage.value = ''
  isSubmitting.value = true

  try {
    await login({
      username: username.value,
      password: password.value,
    })
    router.push(redirectPath.value)
  } catch {
    errorMessage.value = '用户名或密码不正确'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <main class="login-page">
    <section class="login-hero">
      <div class="brand-mark login-brand-mark">T</div>
      <h1>Temu Tools</h1>
      <p>多店铺运营工具台</p>
    </section>

    <section class="login-panel">
      <div>
        <p class="section-label">账号登录</p>
        <h2>进入工作台</h2>
      </div>

      <va-alert v-if="errorMessage" color="danger" dense>
        {{ errorMessage }}
      </va-alert>

      <form class="login-form" @submit.prevent="submitLogin">
        <va-input v-model="username" label="用户名" autocomplete="username" />
        <va-input v-model="password" label="密码" type="password" autocomplete="current-password" />
        <va-button class="login-submit" type="submit" :loading="isSubmitting">
          <LogIn :size="18" />
          登录
        </va-button>
      </form>
    </section>
  </main>
</template>
