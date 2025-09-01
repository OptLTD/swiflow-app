<script setup lang="ts">
import { onMounted, PropType } from 'vue'
import { ref, computed, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: Array as PropType<string[]>,
    default: () => []
  },
  options: {
    type: Array as PropType<OptMeta[]>,
    default: () => []
  },
  grouped: {
    type: Boolean,
    default: false
  },
  label: {
    type: String,
    default: ''
  },
  placeholder: {
    type: String,
    default: '请选择'
  }
})

const emit = defineEmits(['update:modelValue'])
const focused = ref(false)
const selected = ref<string[]>([...props.modelValue])

watch(() => props.modelValue, (val) => {
  selected.value = [...val]
})

const showDropdown = computed(() => {
  return focused.value
})
const width = computed(() => {
  let size = 10;
  props.options.forEach((x) => {
    if (x.label.length > size) {
      size = x.label.length
    }
  })
  return `${size * 0.7}rem`
})

const grouped = computed(() => {
  const result = {} as Record<string, OptMeta[]>
  props.options.forEach((x) => {
    if (!result[x.group]) {
      result[x.group] = [] as OptMeta[]
    }
    result[x.group].push(x)
  })
  return result
})

const displayText = computed(() => {
  if (!selected.value.length) return props.placeholder
  const labels = props.options.filter(opt => {
    return selected.value.includes(opt.value)
  }).map(opt => opt.label)
  if (props.grouped) {
    selected.value.filter(x => x.endsWith(':*')).forEach((key) => {
      labels.push(key)
    })
  }
  if (labels.length === 1) return labels[0]
  if (labels.length > 1) {
    return `${labels[0]}等${labels.length}个`
  }
  return ''
})

const toggleSelect = (item: OptMeta) => {
  // 如果 selected.value 包含 `item.group:*`
  const options = grouped.value[item.group]
  const values = options.map(x => x.value)
  const checked = selected.value.filter(x => {
    return values.includes(x)
  })
  if (selected.value.includes(`${item.group}:*`)) {
    checked.splice(0, checked.length)
    checked.push(...values)
  }
  selected.value = selected.value.filter(x => {
    return !values.includes(x) && `${item.group}:*` !== x
  })

  const idx = checked.indexOf(item.value)
  if (idx === -1 && !item.disabled) {
    checked.push(item.value)
  } else {
    checked.splice(idx, 1)
  }

  if (checked.length == values.length) {
    // 添加当前分组的全选标记（前面已经过滤掉了当前分组的所有选项和全选标记）
    selected.value.push(`${item.group}:*`)
    groupSelected.value[item.group] = true
  } else {
    selected.value.push(...checked)
    groupSelected.value[item.group] = false
  }
  emit('update:modelValue', [...selected.value])
}

const groupSelected = ref<Record<string, boolean>>({})
onMounted(() => {
  groupSelected.value = getGroupSelected()
})


const getGroupSelected = () => {
  const result = {} as Record<string, boolean>
  if (!props.grouped) {
    return result
  }
  const keys = Object.keys(grouped.value)
  keys.forEach((key) => {
    result[key] = selected.value.includes(`${key}:*`)
    if (result[key]) {
      return
    }
    const options = grouped.value[key]
    const items = options.map(x => x.value)
    const checked = selected.value.filter(x => {
      return items.includes(x)
    })
    result[key] = checked.length === items.length
  })
  return result
}

const toggleGroupKey = (key: string) => {
  if (!props.grouped) {
    return
  }
  const options = grouped.value[key] || []
  if (options.length && options[0].disabled) {
    return
  }
  const checked = selected.value.includes(`${key}:*`)
  const values = options.map(x => x.value)
  selected.value = selected.value.filter(x => {
    return !values.includes(x) && `${key}:*` !== x
  })
  // 之前没选则追加
  if (!checked) {
    groupSelected.value[key] = true
    selected.value.push(`${key}:*`)
  } else {
    groupSelected.value[key] = false
  }
  emit('update:modelValue', [...selected.value])
}

const isGroupChecked = (key: string) => {
  return selected.value.includes(`${key}:*`)
}
const isOptionChecked = (item: OptMeta) => {
  const allChecked = selected.value.includes(`${item.group}:*`)
  if (item.group && allChecked) {
    return true
  }
  return selected.value.includes(item.value)
}

const onBlur = () => {
  setTimeout(() => {
    focused.value = false
    emit('update:modelValue', [...selected.value])
  }, 150)
}

const getCheckboxId = (val: string) => `tagselect-${val}`
</script>

<template>
  <div class="formkit-outer">
    <div class="formkit-wrapper">
      <label v-if="label" class="formkit-label">{{ label }}</label>
      <div tabindex="0" :class="{ 'is-focused': focused }"
          class="formkit-input select-input formkit-inner"
          @click="focused = !focused" @blur="onBlur"
        >
          <span class="tag-select-label" :class="{empty: !selected.length}">
            {{ displayText }}
          </span>
          <span class="tag-select-arrow" :class="{open: showDropdown}">▼</span>
          <div v-if="showDropdown" class="dropdown-wrapper" :style="{width}">
            <div
              v-if="!props.grouped" v-for="item in options"
              :key="item.value" class="dropdown-item"
              @click.stop="toggleSelect(item)"
            >
              <input type="checkbox"
                :disabled="item.disabled"
                :id="getCheckboxId(item.value)"
                :checked="isOptionChecked(item)"
                @mouseup.prevent @mousedown.prevent
              />
              <span class="dropdown-label">{{ item.label }}</span>
            </div>
            <template v-else>
              <template v-for="items,group in grouped" :key="group">
                <div class="dropdown-item"
                    @click.stop="toggleGroupKey(group)"
                  >
                  <input type="checkbox"
                    :disabled="items[0].disabled"
                    :checked="isGroupChecked(group)"
                    @mouseup.prevent @mousedown.prevent
                  />
                  <span class="dropdown-group">{{ group }}</span>
                </div>
                <div v-for="item in items"
                    @click.stop="toggleSelect(item)"
                    :key="item.value" class="dropdown-item"
                  >
                  <input type="checkbox"
                    :disabled="item.disabled"
                    :id="getCheckboxId(item.value)"
                    :checked="isOptionChecked(item)"
                    @mouseup.prevent @mousedown.prevent
                  />
                  <span class="dropdown-label">{{ item.label }}</span>
                </div>
              </template>
            </template>
          </div>
        </div>
    </div>
  </div>
</template>

<style scoped>
.formkit-outer, .formkit-wrapper {
  min-width: 12rem;
  max-width: var(--fk-max-width-input);
}
.select-input.formkit-input {
  padding: 10px 14px;
  min-height: 38px;
  display: flex;
  align-items: center;
  position: relative;
  cursor: pointer;
  background: var(--bg-main);
  outline: none;
  font-size: 1em;
  box-sizing: border-box;
}
.select-input>.tag-select-label {
  flex: 1;
  color: var(--color-primary);
  font-size: 1em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.select-input>.tag-select-label.empty {
  color: var(--color-tertiary);
}
.select-input>.tag-select-arrow {
  margin-left: 8px;
  font-size: 1em;
  transition: transform 0.2s;
  color: var(--color-secondary);
  user-select: none;
  display: flex;
  align-items: center;
}
.select-input>.tag-select-arrow.open {
  transform: rotate(180deg);
}
.select-input>.dropdown-wrapper {
  position: absolute;
  left: 0;
  top: 100%;
  width: 100%;
  z-index: 10;
  background: var(--bg-main);
  border: 1.5px solid var(--color-divider);
  border-radius: 8px;
  margin: 4px 0 0 0;
  box-shadow: 0 6px 24px rgba(0,0,0,0.10);
  max-height: 220px;
  overflow-y: auto;
  padding: 8px 0;
}
.select-input .dropdown-group {
  font-weight: bold;
}
.select-input .dropdown-item {
  display: flex;
  align-items: center;
  padding: 6px 16px;
  cursor: pointer;
  user-select: none;
}
.select-input .dropdown-item:hover {
  background: var(--bg-menu);
}
.select-input .dropdown-item input[type="checkbox"] {
  margin-right: 8px;
}
</style>