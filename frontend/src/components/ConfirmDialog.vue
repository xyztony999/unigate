<script setup lang="ts">
import { confirmState, resolveConfirm } from '../lib/confirm'
</script>
<template>
  <Teleport to="body">
    <div v-if="confirmState.open" class="modal-backdrop confirm-backdrop" @click.self="resolveConfirm(false)">
      <section class="confirm-dialog" role="alertdialog" aria-modal="true" :aria-label="confirmState.title">
        <h2>{{ confirmState.title }}</h2>
        <p>{{ confirmState.message }}</p>
        <div class="form-actions">
          <button type="button" class="secondary" @click="resolveConfirm(false)">取消</button>
          <button type="button" :class="{ danger: confirmState.danger }" @click="resolveConfirm(true)">
            {{ confirmState.confirmText }}
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>
