<script setup lang="ts">
/* eslint-disable func-names */
import dayjs from 'dayjs';
import { computed, PropType } from 'vue';

const props = defineProps({
  detail: {
    type: Object as PropType<ActionMsg|null>,
    default: null
  },
})

const timestamp = computed(() => {
  const now = dayjs();
  const target = dayjs(props.detail!.datetime);
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

// const handleFileClick = (e: MouseEvent) => {
//   const ele = e.target as HTMLLinkElement
//   console.log('click', ele.getAttribute('href'))
// }
</script>

<template>
  <ul class="msg-footer">
    <li class="time">{{ timestamp }}</li>
    <li class="flex"></li>
    <li class="btn">
      <icon icon="icon-replay" />
    </li>
  </ul>
</template>

<style scoped>
.msg-footer{
  margin: 5px 0;
  list-style: none;
  margin-top: -5px;

  gap: 5px;
  display: flex;
  flex-direction: row;
  padding-inline-end: 1em;
  padding-inline-start: 1em;
}
.msg-footer>li.flex{
  flex: 1;
  display: flex;
}
.msg-footer>li.btn{
  display: none;
  margin-top: -2px;
}
.msg-footer>li.time {
  font-size: 13px;
  color: var(--color-secondary);
}
.msg-item:hover li.btn{
  display: block;
}
.msg-item:hover li.time {
  color: var(--color-primary)
}
</style>
