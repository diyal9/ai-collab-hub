<template>
  <div class="flex items-center justify-center h-screen bg-gray-100">
    <div class="bg-white p-8 rounded-lg shadow-md w-80">
      <h2 class="text-2xl font-bold mb-4 text-center">AI 协作平台</h2>
      <el-form @submit.prevent="doLogin">
        <el-input v-model="username" placeholder="用户名" class="mb-3" autocomplete="username" />
        <el-input
          v-model="password"
          type="password"
          placeholder="密码"
          class="mb-4"
          show-password
          autocomplete="current-password"
          @keyup.enter="doLogin"
        />
        <el-button type="primary" class="w-full" :loading="loading" @click="doLogin">登录</el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login } from '../api'
import { setAuth } from '../utils/auth'
import { normalizeRedirect } from '../utils/navigation'
import { notifySuccess, showApiError } from '../utils/notify'

const username = ref('')
const password = ref('')
const loading = ref(false)
const router = useRouter()
const route = useRoute()

const doLogin = async () => {
  if (!username.value || !password.value) {
    showApiError(new Error('请输入用户名和密码'))
    return
  }

  loading.value = true
  try {
    const { data } = await login(username.value, password.value)
    setAuth(data.token, data.role)
    notifySuccess('登录成功')
    router.push(normalizeRedirect(typeof route.query.redirect === 'string' ? route.query.redirect : undefined))
  } catch (error) {
    showApiError(error, '登录失败，请检查用户名和密码')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.w-full {
  width: 100%;
}
</style>
