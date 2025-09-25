<script setup lang="ts">
import { PropType, onMounted } from 'vue'
import { ref, watch, computed } from 'vue'

const props = defineProps({
  current: {
    type: String as PropType<string>,
    default: () => ''
  },
  items: {
    type: Array as PropType<MenuMeta[]>,
    default: () => []
  },
  group: {
    type: Array as PropType<MenuMeta[]>,
    default: () => []
  },
  active: {
    type: Object as PropType<MenuMeta>,
    default: () => null
  },
  divide: {
    type: Boolean as PropType<Boolean>,
    default: () => true
  }
})

const active = ref<string>(props.current)
const emit = defineEmits(['click', 'create', 'remove'])

// 用户手动折叠状态管理
const allCollapsed = ref<Set<string>>(new Set())

// 按group分组items
const groupedItems = computed(() => {
  const groups: Record<string, MenuMeta[]> = {}
  
  // 初始化所有group的分组
  props.group.forEach(groupItem => {
    groups[groupItem.value] = []
  })
  
  // 将items按group分组
  props.items.forEach(item => {
    const groupKey = item.group || 'other'
    if (groups[groupKey]) {
      groups[groupKey].push(item)
    } else {
      // 如果group不存在，创建一个"其他"分组
      if (!groups['other']) {
        groups['other'] = []
      }
      groups['other'].push(item)
    }
  })
  
  return groups
})

const clazz = (item: MenuMeta) => {
  return active.value == String(item.value) ? 'active': ''
}

const onClick = (item: MenuMeta) => {
  active.value = String(item.value)
  emit('click', item)
}

const onDelGroup = (groupId: string) => {
  const item = props.group.find(g => {
    return g.value === groupId
  })
  emit('remove', item)
}

const onViewGroup = (groupId: string) => {
  const item = props.group.find(g => {
    return g.value === groupId
  })
  emit('click', item)
}
const onAddSub = (groupId: string) => {
  emit('create', groupId)
}

const onRemove = (item: MenuMeta) => {
  emit('remove', item)
}

const getGroupName = (groupId: string) => {
  const groupItem = props.group.find(g => g.value === groupId)
  return groupItem ? groupItem.label : '其他'
}

// 折叠/展开功能
const toggleCollapse = (groupId: string) => {
  if (allCollapsed.value.has(groupId)) {
    allCollapsed.value.delete(groupId)
  } else {
    allCollapsed.value.add(groupId)
  }
}

const isCollapsed = (groupId: string) => {
  return allCollapsed.value.has(groupId)
}

watch(() => props.current, (val, old) => {
  if (!active.value || val != old) {
    active.value = props.current
  }
})
onMounted(() => {
  Object.keys(groupedItems.value).forEach(groupId => {
    // 如果不是活跃的 group，则折叠
    if (props.active && groupId !== props.active.value) {
      allCollapsed.value.add(groupId)
    }
  })
})
</script>

<template>
  <div class="group-menu">
    <template v-for="(items, groupId) in groupedItems" :key="groupId">
      <div class="group-header" @click="toggleCollapse(groupId)">
        <div class="group-header-left">
          <span class="collapse-icon" :class="{ 
            'collapsed': isCollapsed(groupId)
          }">
            ▼
          </span>
          <span class="group-title">
            {{ getGroupName(groupId) }}
          </span>
          <button v-if="!divide" class="btn-view"
            @click.stop="onViewGroup(groupId)">
            <span class="icon">查看</span>
          </button>
        </div>
        <div class="group-header-right">
          <button v-if="!divide" class="btn-del" 
            @click.stop="onDelGroup(groupId)">
            <span class="icon">x</span>
          </button>
          <button class="btn-add" 
            @click.stop="onAddSub(groupId)" 
            :title="$t('common.addMem')">
            <span class="icon">+</span>
          </button>
        </div>
      </div>
      <div class="menu-list-container" 
        :class="{ 'collapsed': isCollapsed(groupId) }">
        <ul class="menu-list">
          <template v-if="items.length > 0">
            <template v-for="item in items" :key="String(item.value)">
              <li :class="clazz(item)" @click="onClick(item)">
                <slot name="default" :item="item">
                  <span class="item-label">
                    {{ item.label }}
                  </span>
                  <button class="btn-remove" 
                    @click.stop="onRemove(item)" 
                    :title="$t('common.delMemTip')">
                    <span class="icon">×</span>
                  </button>
                </slot>
              </li>
            </template>
          </template>
          <template v-else>
            <li class="empty-item">
              <slot name="empty">
                <span class="empty-label">
                  {{ $t('common.empty') }}
                </span>
              </slot>
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
  border-radius: 4px;
  font-weight: 600;
  font-size: 14px;
  color: #333;
}

.group-header-left {
  display: flex;
  align-items: center;
}

.collapse-icon {
  margin-right: 8px;
  font-size: 12px;
  transition: transform 0.3s ease;
  color: #666;
  cursor: pointer;
}

.collapse-icon.collapsed {
  transform: rotate(-90deg);
}

.group-title {
  flex: 1;
  cursor: pointer;
}

.btn-add,.btn-del,.btn-view {
  background: none;
  border: none;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 16px;
  font-weight: bold;
  outline: none;
  color: var(--color-secondary);
  transition: background-color 0.2s;
}
.btn-view{
  font-size: 13px;
}

.btn-view:hover,
.btn-del:hover,
.btn-add:hover {
  color: var(--color-primary);
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

.menu-list>li{
  margin: 2px 0;
  padding: 0.5em 8px;
  border-radius: 5px;

  display: flex;
  cursor: pointer;
  align-items: center;
  justify-content: space-between;
}

.menu-list > li:hover {
  background: transparent;
  color: var(--text-main);
}

.menu-list > li.active {
  color: var(--text-main);
  background: var(--bg-menu);
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