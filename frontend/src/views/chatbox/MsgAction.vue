<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed, PropType } from 'vue'

const props = defineProps({
  uuid: {
    type: String as PropType<String|Number>,
    default: () => null
  }, 
  item: {
    type: Object as PropType<MsgAct>,
    default: () => null
  }
})

const { t } = useI18n({
  inheritLocale: true,
  useScope: 'global'
})

const title = computed(() => {
  switch (props.item.type) {
    case "complete": {
      return t('common.complete')
    }
    case "execute-command": {
      const act = (props.item as ExecuteCommand)
      return `${t('common.execute')}: ${act.command || ''}`
    }
    case 'path-list-files': {
      const act = (props.item as PathListFiles)
      return `${t('common.listpath')}: ${act.path || ''}`
    }
    case 'file-get-content': {
      const act = (props.item as FileGetContent)
      return `${t('common.viewfile')}: ${act.path || ''}`
    }
    case 'file-replace-text':
    case 'file-put-content': {
      const act = (props.item as FilePutContent)
      return `${t('common.writefile')}: ${act.path || ''}`
    }
    case "use-mcp-tool": {
      const act = (props.item as UseMcpTool)
      return `${t('common.usemcp')}: ${act.desc}`
    }
    case "use-builtin-tool": {
      const act = (props.item as UseBuiltinTool)
      return `${t('common.usetool')}: ${act.desc}`
    }
  }
})
</script>

<template>
  <div class="act-card" @click="$emit('click')">
    {{ title }}
  </div>
</template>
