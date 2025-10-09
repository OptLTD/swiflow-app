<script setup lang="ts">
import MsgDetail from './MsgDetail.vue'
import MsgHeader from './MsgHeader.vue'
defineProps(['messages', 'errmsg', 'loading'])
defineEmits(['check', 'replay', 'display'])
</script>
<template>
    <div class="error-tips" v-if="errmsg">
      <div class="error-detail">{{ errmsg }}</div>
    </div>
    <ul v-if="messages.length > 0" class="msg-list">
      <MsgDetail
        v-for="(item, i) in messages"
        :key="i" :detail="item"
        @check="$emit('check', $event)"
        @replay="$emit('replay', $event)"
        @display="$emit('display', $event)"
        :is-last="i+1 === messages.length"
      >
        <template #header>
          <MsgHeader :detail="item"/>
        </template>
      </MsgDetail>
      <template v-if="loading && loading!.actions">
      <MsgDetail :detail="loading" :loading="true">
        <template #header v-if="messages.length == 1">
          <MsgHeader :detail="loading"/>
        </template>
      </MsgDetail>
      </template>
    </ul>
    <slot v-else>
      <div class="empty-result">
        {{ $t('common.empty') }}
      </div>
    </slot>
</template>

<style scoped>
.error-tips {
  top: -10px;
  z-index: 1;
  margin: 0px -10px;
  padding: 10px 10px;
  display: flex;
  position: sticky;
  background-color: #fff;
  flex-direction: column;
  position: -webkit-sticky;
  width: -webkit-fill-available;
}
.error-detail {
  padding: 12px 12px;
  border-radius: 5px;
  color: orangered;
  font-size: 1.1rem;
  font-weight: bold;
  text-align: center;
  background-color: #fff4f4;
}
.msg-list {
  display: flex;
  min-height: 100px;
  flex-direction: column;
  margin-block-end: 0;
  margin-block-start: 0;
  padding-inline-start: 18px;
}
.msg-item {
  display: flex;
  max-width: 100%;
  flex-direction: column;
}
.empty-result{
  height: calc(100vh - 260px);
}
</style> 