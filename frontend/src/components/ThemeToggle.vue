<script setup lang="ts">
import { theme, toggleTheme } from '../lib/theme'
</script>
<template>
  <button type="button" class="theme-toggle" @click="toggleTheme"
    :aria-label="theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'">
    {{ theme === 'dark' ? '浅色模式' : '深色模式' }}
  </button>
</template>
