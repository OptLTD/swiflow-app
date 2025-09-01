<script setup lang="ts">
import dayjs from 'dayjs';
import { Tippy } from 'vue-tippy';
import * as emoji from 'node-emoji';
import { md } from '@/support/index';
import { computed, PropType } from 'vue';
import { showMyBotForm } from '@/logics/index'
import { usePreferredDark } from '@vueuse/core';
import { showContext } from '@/logics/popup'


const props = defineProps({
  detail: {
    type: Object as PropType<ActionMsg|null>,
    default: null
  },
  current: {
    type: Object as PropType<BotEntity|null>,
    default: null
  },
})

const timestamp = computed(() => {
  const now = dayjs();
  const { datetime } = props.detail || {}
  const target = dayjs(datetime || Date.now());
  const diffDays = now.diff(target.startOf('day'), 'day');
  
  if (diffDays === 0) {
    return target.format('h:mm A'); // 今天: "2:30 PM"
  } else if (diffDays === 1) {
    return 'Yesterday ' + target.format('h:mm A'); // 昨天
  } else if (diffDays < 7) {
    return target.format('dddd'); // 本周: "Monday"
  } else if (diffDays < 365) {
    return target.format('MMM D'); // 今年: "Jun 30"
  } else {
    return target.format('MMM D, YYYY'); // 往年: "Jun 30, 2020"
  }
})

const thinking = computed(() => {
  const { thinking } = props.detail || {}
  return thinking ? md.render(thinking) : ''
})

const context = computed(() => {
  const context = props.detail?.context  as any
  if (!context || !context.context) {
    return ''
  }
  return context.context ? md.render(context.context) : ''
})

const theTheme = computed(() => {
  const isDark = usePreferredDark().value
  return isDark ? 'dark' : 'light'
})

const showModifyBot = () => {
  const info = props.current
  if (!info?.uuid) {
    return
  }
  return showMyBotForm(info)
}
const showContextModal = () => {
  showContext(props.detail?.context || {})
}
</script>

<template>
  <div @click="showModifyBot" class="msg-avatar">
    {{ emoji.get(current?.emoji || 'man_technologist') }}
  </div>
  
  <ul class="msg-header">
    <li class="time">{{ timestamp }}</li>
    <li class="btn">
      <tippy v-if="thinking" :theme="theTheme" trigger="mouseenter click">
        <button class="btn-icon icon-small btn-thought"/>
        <template #content>
          <div class="thinking" v-html="thinking"/>
        </template>
      </tippy>
      <button v-if="context" 
        @click="showContextModal"
        class="btn-icon icon-small btn-todolist" 
      />
    </li>
  </ul>
</template>

<style scoped>
.msg-avatar {
  left: 8px;
  width: 24px;
  height: 24px;
  font-size: 24px;
  line-height: 24px;
  border-radius: 18px;
  position: absolute;
  text-align: center;
  cursor: pointer;
}
.msg-header {
  margin: 0px 0px;
  list-style: none;

  gap: 5px;
  display: flex;
  flex-direction: row;
  padding-inline-end: 1em;
  padding-inline-start: 1em;
}
.thinking{
  margin: 8px 8px;
}
.thinking:deep(:last-child),
.thinking:deep(:first-child){
  margin-block-end: 0em;
  margin-block-start: 0em;
}
</style>
