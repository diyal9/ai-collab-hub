<template>
  <div class="flex flex-col items-center justify-center min-h-[60vh] text-center">
    <div class="text-6xl font-bold text-gray-300 mb-4">404</div>
    <h1 class="text-xl font-semibold text-gray-800 mb-2">页面不存在</h1>
    <p class="text-gray-500 mb-6">您访问的路径可能已变更或尚未开放。</p>
    <router-link
      to="/"
      class="inline-flex items-center px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition"
    >
      返回概览
    </router-link>
  </div>
</template>
