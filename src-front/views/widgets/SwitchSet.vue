<script setup lang="ts">
import { Tippy } from 'vue-tippy'
import { useI18n } from 'vue-i18n';
import { useAppStore } from '@/stores/app'
import { showUseModelPopup } from '@/logics';

const { t } = useI18n({
  inheritLocale: true,
  useScope: 'global'
})

const app = useAppStore()

const items = [
  { uuid: 'set-bot', icon: 'btn-robot', name: t('menu.botSet') },
  { uuid: 'set-mcp', icon: 'btn-tools', name: t('menu.mcpSet') },
  { uuid: 'set-mem', icon: 'btn-memory', name: t('menu.memSet') },
  { uuid: 'set-todo', icon: 'btn-clock', name: t('menu.todoSet') },
  { uuid: 'set-model', icon: 'btn-switch', name: t('menu.modelSet') },
  { uuid: 'setting', icon: 'btn-gear', name: t('menu.basicSet') },
]

const onSwitchSet = (item: any) => {
  switch (item.uuid) {
    case 'set-model':
      showUseModelPopup(app.getAuthGate)
      break
    case 'setting': {
      app.setContent(true)
      app.setChatBar(false)
      app.setAction('setting')
      document.title = 'Setting'
      break
    }
    case 'set-mcp': {
      app.setContent(true)
      app.setChatBar(false)
      app.setAction('set-mcp')
      document.title = 'Set Mcp'
      break
    }
    case 'set-bot': {
      app.setContent(true)
      app.setChatBar(false)
      app.setAction('set-bot')
      document.title = 'Set Bot'
      break
    }
    case 'set-mem': {
      app.setContent(true)
      app.setChatBar(false)
      app.setAction('set-mem')
      document.title = 'Set Mem'
      break
    }
    case 'set-todo': {
      app.setContent(true)
      app.setChatBar(false)
      app.setAction('set-todo')
      document.title = 'Set Todo'
      break
    }
  }
}
</script>

<template>
  <tippy interactive :theme="app.getTheme" arrow 
    placement="left-start" trigger="mouseenter click">
    <button class="btn-icon icon-large btn-gear"/>
    <template #content>
      <template v-for="item in items">
        <div class="opt-set" @click="onSwitchSet(item)">
          <a class="btn-icon icon-mini" :class="item.icon" />
          <span>{{ item.name }}</span>
        </div>
      </template>
    </template>
  </tippy>
</template>

<style scoped>
.opt-set{
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
.opt-set.disabled{
  cursor: not-allowed;
  color: var(--color-tertiary);
}
.opt-set:hover{
  box-shadow: inset 1px;
}
.opt-set:first-of-type{
  margin-top: 5px;
}
.opt-set:last-of-type{
  border-bottom: 0;
  margin-bottom: 5px;
}
.opt-set.checked{
  font-weight: bold;
}
.opt-set>.btn-icon{
  width: 1rem;
  height: 1rem;
  margin-right: 5px;
}
.opt-set>.btn-icon:hover{
  background-color: unset;
}
</style>