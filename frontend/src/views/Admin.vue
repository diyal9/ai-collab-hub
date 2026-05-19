<template>
  <div class="space-y-4">
    <h1 class="text-2xl font-bold">⚙️ 系统管理 (Admin Only)</h1>
    <div class="bg-white p-4 rounded shadow">
      <h3 class="font-bold mb-2">添加用户 / AI 终端</h3>
      <div class="flex gap-2">
        <el-input v-model="newUser.name" placeholder="用户名" />
        <el-input v-model="newUser.pass" placeholder="初始密码" />
        <el-button @click="addUser" type="primary">创建</el-button>
      </div>
    </div>
    <el-table :data="users" class="w-full">
      <el-table-column prop="username" label="用户名" />
      <el-table-column prop="role" label="角色" />
      <el-table-column prop="token" label="API/MCP Token" />
    </el-table>
  </div>
</template>
<script setup>
import { ref, onMounted } from 'vue'
import { getUsers, createUser } from '../api'
const users = ref([]), newUser = ref({ name: '', pass: '' })
onMounted(async () => { users.value = (await getUsers()).data })
const addUser = async () => { await createUser(newUser.value.name, newUser.value.pass); users.value = (await getUsers()).data }
</script>
