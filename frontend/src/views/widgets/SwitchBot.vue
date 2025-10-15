<script setup lang="ts">
import { PropType } from 'vue'
import { Tippy } from 'vue-tippy'
import { useAppStore } from '@/stores/app'

const app = useAppStore()
const props = defineProps({
  disabled: {
    type: Boolean as PropType<Boolean>,
    default: () => null,
  }
})

const checked = (bot: BotEntity) => {
  return app.getActive?.uuid == bot.uuid
}

const emit = defineEmits(['click'])
const doClick = (bot: BotEntity) => {
  if (props.disabled) {
    return
  }
  emit('click', bot)
}

</script>

<template>
  <tippy interactive :theme="app.getTheme" arrow 
    placement="top-start" trigger="mouseenter click">
    <icon icon="icon-robot" :text="app.getActive?.name"/>
    <template #content>
      <template v-for="item in app.getBotList">
        <div class="opt-bot checked" :class="{disabled}" v-if="checked(item)">
          <icon icon="icon-robot" :text="item.name" size="mini"/>
        </div>
        <div v-else class="opt-bot no-checked" :class="{disabled}" @click="doClick(item)">
          <icon icon="icon-none" :text="item.name" size="mini"/>
        </div>
      </template>
    </template>
  </tippy>
</template>

<style scoped>
.opt-bot{
  font-size: 13px;
  cursor: pointer;
  margin: 0px 5px;
  padding: 5px 0px;
  border-width: 0;
  border-style: dotted;
  border-bottom-width: 1px;
  border-color: #d5d5d5;
  min-width: 6rem;
  display: flex;
  align-items: center;
  width: -webkit-fill-available;
}
.opt-bot.disabled{
  cursor: not-allowed;
  color: var(--color-tertiary);
}
.opt-bot:hover{
  box-shadow: inset 1px;
}
.opt-bot:first-of-type{
  margin-top: 5px;
}
.opt-bot:last-of-type{
  border-bottom: 0;
  margin-bottom: 5px;
}
.opt-bot.checked{
  font-weight: bold;
}
</style>