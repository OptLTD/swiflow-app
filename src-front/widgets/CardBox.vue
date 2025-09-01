
<script setup lang="ts">
import {ref} from 'vue'

const props = defineProps({
  title: {
    type: String,
    default: () => ''
  },
  fold: {
    type: Boolean,
    default: () => false
  },
})
const fold = ref(props.fold)
</script>

<template>
  <div class="fold-card" :class="{fold}">
    <div class="fold-head">
      <div class="leading" @click="fold=!fold">
        {{ props.title }}
      </div>
      <div class="actions">
        <slot name="action"/>
      </div>
    </div>
    <div v-if="!fold" class="fold-body">
      <slot name="default"/>
    </div>
  </div>
</template>

<style scoped>
.fold-card {
  margin-bottom: 1em;
  border-radius: 5px;
  border: 1px solid #ddd;
}
.fold-head{
  display: flex;
  padding: 5px 5px;
  border-bottom: 1px solid #ddd;
  justify-content: space-between;
}
.fold-body {
  font-size: 1rem;
  margin: -0.8em 0;
}
.fold .fold-head{
  border-bottom: none
}



.leading {
  cursor: pointer;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  position: relative;
  padding-left: 2em;
  font-size: 1rem;
  height: 22px;
}

.leading::before{
  left: 2px;
  width: 22px;
  height: 22px;
  content: " ";
  position: absolute;
  margin-right: 5px;
  display: inline-block;
  background-size: cover;
  background-image: url("/assets/fold-show.svg");
}
.fold .leading::before{
  background-image: url("/assets/fold-hide.svg");
}
</style>
