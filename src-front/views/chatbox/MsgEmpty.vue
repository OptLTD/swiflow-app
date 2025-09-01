<script setup lang="ts">
const suggestions = [
  '帮我看一下孙燕姿最近的演唱会都有哪些？',
  '帮我设计一个俄罗斯方块',
  '帮我从数据库分析下最近的订单，看下那些事紧急要处理的',
  '帮我做一个贪吃蛇小游戏',
]
</script>

<template>
  <div class="empty-result">
    <h3>{{ $t('common.tryAsk') }}</h3>
    <ul class="suggestions">
      <template v-for="item in suggestions">
      <li @click="$emit('select', item)">{{ item }}</li>
      </template>
    </ul>
  </div>
</template>

<style scoped>
.suggestions {
  list-style-type: none;
  width: -webkit-fill-available;
  display: flex;
  flex-wrap: wrap; /* 关键属性：允许换行 */
  list-style: none; /* 移除默认列表样式 */
  padding: 0;
  margin: 0;
  gap: 10px; /* 项目间距 */
}
.suggestions > li{
  cursor: pointer;
  padding: 10px 15px;
  background: #f5f5f5;
  border-radius: 5px;
  flex: 0 0 calc(50% - 10px); /* 三列布局，考虑gap间距 */
  box-sizing: border-box; /* 确保宽度计算包含padding和border */
}
</style>
