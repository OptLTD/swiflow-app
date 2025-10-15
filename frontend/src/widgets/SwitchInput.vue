<template>
  <div class="switch-container" :class="`switch-${size}`">
    <input
      :id="id"
      type="checkbox"
      class="switch-input"
      :checked="modelValue"
      :disabled="disabled"
      @change="onInputChange"
    />
    <label :for="id" class="switch-label"></label>
  </div>
</template>

<script setup lang="ts">
defineProps({
  id: { type: String, default: 'switch' },
  size: { type: String, default: 'small' },
  disabled: { type: Boolean, default: false },
  modelValue: { type: Boolean, required: true },
})
const emit = defineEmits(['update:modelValue', 'change'])

function onInputChange(e: Event) {
  const checked = (e.target as HTMLInputElement)?.checked
  emit('update:modelValue', checked)
  emit('change', checked)
}
</script>

<style scoped>
.switch-container {
  display: flex;
  align-items: center;
  height: 24px;
}
.switch-input {
  display: none;
}
.switch-label {
  width: 40px;
  height: 20px;
  background: var(--color-tertiary);
  border-radius: 20px;
  position: relative;
  cursor: pointer;
  transition: background 0.2s;
  display: inline-block;
}
[data-theme="dark"] .switch-label {
  background: var(--color-tertiary);
}
.switch-label::after {
  content: "";
  position: absolute;
  left: 2px;
  top: 2px;
  width: 16px;
  height: 16px;
  background: var(--bg-main);
  border-radius: 50%;
  transition: left 0.2s;
}
.switch-input:checked + .switch-label {
  background: #4caf50;
}
.switch-input:checked + .switch-label::after {
  left: 22px;
}
.switch-input:disabled + .switch-label {
  background: #ccc;
  cursor: not-allowed;
  opacity: 0.6;
}
.switch-input:disabled + .switch-label::after {
  background: #eee;
}

/* small 尺寸 */
.switch-small {
  height: 16px;
}
.switch-small .switch-label {
  width: 28px;
  height: 14px;
}
.switch-small .switch-label::after {
  width: 10px;
  height: 10px;
  left: 2px;
  top: 2px;
}
.switch-small .switch-input:checked + .switch-label::after {
  left: 14px;
}
</style> 