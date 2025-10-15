<script setup lang="ts">
import { ref } from 'vue'
import { FormKit } from '@formkit/vue'

const formModel = ref({ 
  servers: `{
  "mcpServers": {
    "fetch": {
      "command": "uvx",
      "args": ["mcp-server-fetch"]
    }
  }
}`})

const getFormModel = () => {
  return formModel.value
}

defineExpose({ getFormModel })
</script>

<template>
  <FormKit type="form"
    v-model="formModel"
    :actions="false">
    <FormKit
      type="textarea"
      name="servers"
      label="Mcp Servers"
      outer-class="new-mcp"
      v-model="formModel.servers"
      help="paste config here"
    />
  </FormKit>
</template>

<style scoped>
@import url('@/styles/form.css');
.title{
  text-align: center;
  margin-top: -15px;
  margin-bottom: 15px;
}
/* 调整FormKit textarea高度 */
.new-mcp :deep(textarea) {
  resize: vertical;
  padding: 2px !important;
  min-height: 22.5rem !important;
}
</style>
