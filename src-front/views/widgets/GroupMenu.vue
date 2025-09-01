<script setup lang="ts">
import { PropType, ref, watch, computed } from 'vue'

const props = defineProps({
  current: {
    type: String as PropType<string>,
    default: () => ''
  },
  items: {
    type: Array as PropType<MemEntity[]>,
    default: () => []
  },
  bots: {
    type: Array as PropType<BotEntity[]>,
    default: () => []
  },
  active: {
    type: Object as PropType<BotEntity>,
    default: () => null
  }
})

const active = ref<string>(props.current)
const emit = defineEmits(['click', 'create', 'remove'])

// 折叠状态管理 - 默认展开活跃 Bot 的记忆，折叠其他的
const collapsedGroups = computed(() => {
  const collapsed = new Set<string>()
  
  // 遍历所有 bot 分组
  Object.keys(groupedItems.value).forEach(botId => {
    // 如果不是活跃的 bot，则折叠
    if (props.active && botId !== props.active.uuid) {
      collapsed.add(botId)
    }
  })
  
  // 添加用户手动折叠的分组
  userCollapsedGroups.value.forEach(botId => {
    collapsed.add(botId)
  })
  
  return collapsed
})

// 用户手动折叠状态管理
const userCollapsedGroups = ref<Set<string>>(new Set())

// 按bot分组items
const groupedItems = computed(() => {
  const groups: Record<string, MemEntity[]> = {}
  
  // 初始化所有bot的分组
  props.bots.forEach(bot => {
    groups[bot.uuid] = []
  })
  
  // 将items按bot分组
  props.items.forEach(item => {
    if (groups[item.bot]) {
      groups[item.bot].push(item)
    } else {
      // 如果bot不存在，创建一个"其他"分组
      if (!groups['other']) {
        groups['other'] = []
      }
      groups['other'].push(item)
    }
  })
  
  return groups
})

const clazz = (item: MemEntity) => {
  return active.value == String(item.id) ? 'active': ''
}

const onClick = (item: MemEntity) => {
  active.value = String(item.id)
  emit('click', item)
}

const onCreate = (botId: string) => {
  emit('create', botId)
}

const onRemove = (item: MemEntity) => {
  emit('remove', item)
}

const getBotName = (botId: string) => {
  const bot = props.bots.find(b => b.uuid === botId)
  return bot ? bot.name : '其他'
}

// 折叠/展开功能
const toggleCollapse = (botId: string) => {
  if (userCollapsedGroups.value.has(botId)) {
    userCollapsedGroups.value.delete(botId)
  } else {
    userCollapsedGroups.value.add(botId)
  }
}

const isCollapsed = (botId: string) => {
  return collapsedGroups.value.has(botId)
}

watch(() => props.current, (val, old) => {
  if (!active.value || val != old) {
    active.value = props.current
  }
})

// 监听活跃 Bot 变化，清除用户手动折叠状态
watch(() => props.active, () => {
  userCollapsedGroups.value.clear()
})
</script>

<template>
  <div class="group-menu">
    <template v-for="(items, botId) in groupedItems" :key="botId">
      <div class="group-header" @click="toggleCollapse(botId)">
        <div class="group-header-left">
          <span class="collapse-icon" :class="{ 'collapsed': isCollapsed(botId) }">
            ▼
          </span>
          <span class="group-title">{{ getBotName(botId) }}</span>
        </div>
        <button class="btn-add" 
          @click.stop="onCreate(botId)" 
          :title="$t('common.addMem')">
          <span class="icon">+</span>
        </button>
      </div>
      <div class="menu-list-container" 
        :class="{ 'collapsed': isCollapsed(botId) }">
        <ul class="menu-list">
          <template v-if="items.length > 0">
            <template v-for="item in items" :key="String(item.id)">
              <li :class="clazz(item)" @click="onClick(item)">
                <slot name="default" :item="item">
                  <span class="item-label">
                    {{ item.data.substring(0, 12) }}
                  </span>
                </slot>
                <button class="btn-remove" 
                  @click.stop="onRemove(item)" 
                  :title="$t('common.delMemTip')">
                  <span class="icon">×</span>
                </button>
              </li>
            </template>
          </template>
          <template v-else>
            <li class="empty-item">
              <span class="empty-label">
                {{ $t('common.noMem') }}
              </span>
            </li>
          </template>
        </ul>
      </div>
    </template>
  </div>
</template>

<style scoped>
.group-menu {
  width: 100%;
}

.group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  margin: 8px 0 4px 0;
  background-color: #f5f5f5;
  border-radius: 4px;
  font-weight: 600;
  font-size: 14px;
  color: #333;
  cursor: pointer;
  transition: background-color 0.2s;
}

.group-header:hover {
  background-color: #e9ecef;
}

.group-header-left {
  display: flex;
  align-items: center;
  flex: 1;
}

.collapse-icon {
  margin-right: 8px;
  font-size: 12px;
  transition: transform 0.3s ease;
  color: #666;
}

.collapse-icon.collapsed {
  transform: rotate(-90deg);
}

.group-title {
  flex: 1;
}

.btn-add {
  background: none;
  border: none;
  color: #007bff;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 16px;
  font-weight: bold;
  transition: background-color 0.2s;
}

.btn-add:hover {
  background-color: #e3f2fd;
}

.menu-list-container {
  overflow: hidden;
  transition: max-height 0.3s ease, opacity 0.3s ease;
  max-height: 1000px;
  opacity: 1;
}

.menu-list-container.collapsed {
  max-height: 0;
  opacity: 0;
}

.menu-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.menu-list > li {
  height: 24px;
  margin: 2px 0;
  padding: 0.5em 0.5em;
  line-height: 24px;
  border-radius: 5px;
  display: flex;
  cursor: pointer;
  align-items: center;
  justify-content: space-between;
  font-weight: bold;
  transition: background-color 0.2s;
  position: relative;
}

.menu-list > li:hover {
  background-color: #f8f9fa;
}

.menu-list > li.active {
  background-color: #007bff;
  color: white;
}

.empty-item {
  height: 24px;
  margin: 2px 0;
  padding: 0.5em 1.2em;
  line-height: 24px;
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: normal;
  color: #999;
  font-style: italic;
  cursor: default;
  background-color: transparent;
}

.empty-item:hover {
  background-color: transparent;
}

.empty-label {
  flex: 1;
  text-align: center;
  font-size: 12px;
}

.item-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btn-remove {
  right: 8px;
  position: absolute;
  background: none;
  border: none;
  color: #dc3545;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 16px;
  font-weight: bold;
  opacity: 0;
  transition: opacity 0.2s, background-color 0.2s;
}

.menu-list > li:hover .btn-remove {
  opacity: 1;
}

.btn-remove:hover {
  background-color: #f8d7da;
}

.icon {
  display: inline-block;
  line-height: 1;
}
</style> 