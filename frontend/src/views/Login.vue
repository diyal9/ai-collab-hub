<template>
  <div class="flex items-center justify-center h-screen bg-gray-100">
    <div class="bg-white p-8 rounded-lg shadow-md w-80">
      <h2 class="text-2xl font-bold mb-4 text-center">AI 协作平台</h2>
      <input v-model="u" placeholder="用户名" class="w-full p-2 border rounded mb-2">
      <input v-model="p" type="password" placeholder="密码" class="w-full p-2 border rounded mb-4">
      <button @click="doLogin" class="w-full bg-blue-600 text-white p-2 rounded hover:bg-blue-700">登录</button>
    </div>
  </div>
</template>
<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api'
const u = ref(''), p = ref('')
const router = useRouter()
const doLogin = async () => {
  try {
    const { data } = await login(u.value, p.value)
    localStorage.setItem('token', data.token)
    localStorage.setItem('role', data.role)
    router.push('/')
  } catch (e) { alert('登录失败') }
}
</script>
