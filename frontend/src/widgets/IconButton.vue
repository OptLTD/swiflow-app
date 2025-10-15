<script setup lang="ts">
import { computed } from 'vue'
import { PropType } from 'vue'

const props = defineProps({
  size: {
    type: String as PropType<'mini' | 'small' | 'medium' | 'large'>,
    default: 'medium'
  },
  icon: {
    type: String,
    default: ''
  },
  color: {
    type: String,
    default: ''
  },
  class: {
    type: String,
    default: ''
  },
  text: {
    type: String,
    default: ''
  },
  title: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['click'])

const onClick = (e: MouseEvent) => emit('click', e)

const iconSize = computed(() => {
  switch (props.size) {
    case 'mini': return 16
    case 'small': return 20
    case 'large': return 32
    case 'medium':
    default: return 24
  }
})

const styles = computed(() => ({
  '--icon-size': `${iconSize.value}px`,
  color: props.color || 'var(--icon-color)',
}))

const iconSrc = computed(() => {
  return props.icon ? `/icons/${props.icon}.svg` : ''
})

const iconMaskStyle = computed(() => {
  const url = iconSrc.value
  if (!url) return {}
  return {
    WebkitMaskImage: `url(${url})`,
    maskImage: `url(${url})`,
    WebkitMaskRepeat: 'no-repeat',
    maskRepeat: 'no-repeat',
    WebkitMaskPosition: 'center',
    maskPosition: 'center',
    WebkitMaskSize: 'contain',
    maskSize: 'contain',
    backgroundColor: props.color || 'currentColor'
  }
})
</script>

<template>
  <button type="button" class="icon-button" @click="onClick"
    :class="props.class" :style="styles" :title="title || text">
    <span class="icon-wrap">
      <template v-if="iconSrc">
        <span class="icon-mask" 
          aria-hidden="true" 
          :style="iconMaskStyle" 
        />
      </template>
    </span>
    <span v-if="text" class="label">{{ text }}</span>
  </button>
</template>

<style scoped>
.icon-button {
  border: none;
  outline: none;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 3px;
  border-radius: 5px;
  color: var(--icon-color);
  background: transparent;
  transition: background-color 0.2s ease;
}
.icon-button:hover {
  background-color: var(--bg-menu);
}
.icon-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: var(--icon-size);
  height: var(--icon-size);
}
.icon {
  width: var(--icon-size);
  height: var(--icon-size);
  fill: none;
}
.icon-mask {
  width: var(--icon-size);
  height: var(--icon-size);
  display: inline-block;
}
.icon-img {
  width: var(--icon-size);
  height: var(--icon-size);
}
.label {
  font-size: 13px;
  width: max-content;
}
</style>