<script setup lang="ts">
import { watch, computed } from 'vue'
import { useAppStore } from '@/stores/app'
import { useTaskStore } from '@/stores/task'
import { confirm, request } from '@/support'

import SwitchBot from './widgets/SwitchBot.vue';
import SwitchSet from './widgets/SwitchSet.vue';

import { useI18n } from 'vue-i18n';

const { t } = useI18n({
  inheritLocale: true,
  useScope: 'global'
})
const app = useAppStore()
const task = useTaskStore()

const doToggleMenu = () => {
  app.toggleMenuBar()
}

const doSwitchTask = (item: TaskEntity | null) => {
  const isChatView = ['brower', 'default'].includes(app.getAction)
  const isCurrChat = item && item.uuid == task.getActive
  if (isCurrChat && isChatView) {
    return
  }
  if (!item?.uuid) {
    app.setContent(false)
    document.title = 'New Task'
  }
  app.setChatBar(true)
  task.setActive(item?.uuid || '')
  !isChatView && app.setAction('default')
  item?.name && (document.title = item.name)
}

const doStartProgram = async(uuid: string, act: string) => {
  try {
    const url = `/program?uuid=${uuid}&act=${act}`
    const resp = await request.post(url) as any
    console.log('start server success', resp, uuid)
  } catch (err) {
    console.error('use bot:', err)
  }
}

const doRemoveTask = async(uuid: string) => {
  const msg = t('tips.delTaskMsg')
  // const tip = t('tips.delTaskTip')
  const answer = await confirm(msg);
  if (!answer) {
    return
  }
  try {
    const url = `/task?act=del-task&uuid=${uuid}`
    const resp = await request.post(url) as BotEntity
    console.log('do del-task success', resp, uuid)
  } catch (err) {
    console.error('del-task:', err)
  } finally {
    task.setActive('')
  }
}

const onSwitchBot = (bot: BotEntity) => {
  !task.getActive && setActiveBot(bot)
}


const setActiveBot = async (bot: BotEntity) => {
  try {
    const url = `/bot?act=use-bot&uuid=${bot.uuid}`
    const resp = await request.post(url) as BotEntity
    if (!(resp as any)?.errmsg) {
      app.setActive(resp)
    }
    // console.info('use bot:', resp)
  } catch (err) {
    console.error('use bot:', err)
  }
}

const clazz = (item: TaskEntity) => {
  return item.uuid== task.getActive
    ? 'active' : ''
}

const disabled = computed(() => {
  return task.getActive != ''
})

watch(() => task.getActive, (val) => {
  !val && doSwitchTask(null)
})
</script>

<template>
  <div id="menu-header">
    <button class="btn-icon icon-large btn-menubar" @click="() => doToggleMenu()"/>
    <button class="btn-icon icon-large btn-newchat" @click="() => doSwitchTask(null)"/>
  </div>
  <div id="menu-title">
      <img src="/assets/icon.svg">
      <h1> {{ "SWIFLOW" }} </h1>
  </div>
  <div id="menu-container">
    <dl class="menu-list">
      <template v-if="task.getHistory.length==0">
        <div class="empty-result">
          {{ $t('common.empty') }}
        </div>
      </template>
      <template v-for="item in task.getHistory" :key="item.uuid">
        <dd :class="clazz(item)" @click="doSwitchTask(item)">
          <label>{{ item.name }}</label>
          <button class="btn-icon btn-run" 
            v-if="item.command && !item.process"
            @click.stop="doStartProgram(item.uuid, 'start')"
          />
          <button class="btn-icon btn-stop" 
            v-if="item.command && item.process"
            @click.stop="doStartProgram(item.uuid, 'stop')"
          />
          <button class="btn-icon btn-remove" 
            @click.stop="doRemoveTask(item.uuid)"
          />
        </dd>
      </template>
      <dt class="menu-footer">
        <SwitchBot  
          @click="onSwitchBot" 
          :disabled="disabled"
        />
        <SwitchSet/>
      </dt>
    </dl>
  </div>
</template>

<style scoped>
#menu-container {
  flex-grow: 1;
  overflow-y: auto;
  overflow-x: hidden;
  border-radius: 5px;
  padding: 0px 10px;
  position: relative;
}
.menu-list{
  margin: -2px 0px 0 0px;
  padding-inline-start: 0px;
}
.menu-list>dd{
  cursor: pointer;
  margin: 2px 0px;
  padding: 0px 8px;
  line-height: 32px;
  border-radius: 5px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  align-items: stretch;
  position: relative;
  width: -webkit-fill-available;
}
.menu-list>dd>label{
  cursor: pointer;
}

.menu-list .menu-header {
  display: none;
  margin-bottom: 15px;
}
.menu-list .menu-header>input {
  flex: 1;
  height: 30px;
  border-width: 0px;
  border-radius: 5px;
  margin-right: 5px;
  margin-left: 5px;
  background-color: #d5d5d5;
}
.menu-list .menu-header>input:focus {
  outline-width: 1px;
  outline-style: double;
}
.menu-list .btn-robot {
  position: relative;
  background-size: 100%;
}
.menu-list>dd .btn-run,
.menu-list>dd .btn-stop,
.menu-list>dd .btn-remove {
  display: none;
  min-width: 24px;
  min-height: 24px;
  position: absolute;
  right: 4px; top:4px;
}

.menu-list>dd.active{
  padding-right: 50px;
}
.menu-list>dd.active:hover .btn-run,
.menu-list>dd.active:hover .btn-stop{
  right: 28px;
  display: block;
}
.menu-list>dd.active:hover .btn-remove{
  display: inline-flex;
}
.menu-footer {
  display: flex;
  left: 0px;
  bottom: 10px;
  padding: 0 10px;
  position: absolute;
  width: -webkit-fill-available;
  justify-content: space-between;
}
.menu-footer .btn-robot{
  width: auto;
  height: 24px;
  padding: 8px 5px;
  border-radius: 5px;
  flex-direction: row;
  align-items: center;
  display: inline-flex;
}
.menu-footer .btn-robot::before{
  height: 24px; width: 24px;
  margin-left: -5px;
}
</style>
