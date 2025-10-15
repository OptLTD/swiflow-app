<script setup lang="ts">
import * as emoji from 'node-emoji';
import { onMounted, computed } from 'vue'
import { request, toast } from '@/support'
import { useAppStore } from '@/stores/app'
import { useTaskStore } from '@/stores/task'

import SwitchSet from './widgets/SwitchSet.vue';
import ProfileSet from './widgets/ProfileSet.vue';

const app = useAppStore()
const task = useTaskStore()

const onSwitchBot = (bot: BotEntity) => {
  if (!['brower', 'default'].includes(app.getAction)) {
    app.setChatBar(true)
    app.setContent(false)
    app.setAction('default')
    var active = getRunning.value(bot)
    task.setActive(active?.uuid || '')
  }
  if (app.getActive?.uuid == bot.uuid) {
    return
  }
  setActiveBot(bot)
}


const setActiveBot = async (bot: BotEntity) => {
  try {
    const url = `/bot?act=use-bot&uuid=${bot.uuid}`
    const resp = await request.post<any>(url)
    if (resp?.errmsg) {
      toast.error(resp.errmsg)
      return
    }
    var active = getRunning.value(resp)
    task.setActive(active?.uuid || '')
    app.setActive(resp as BotEntity)
    console.info('use bot:', resp)
  } catch (err) {
    console.error('use bot:', err)
  }
}

const clazz = (item: BotEntity) => {
  const { uuid } = app.getActive || {}
  return item.uuid == uuid ? 'active' : ''
}

const getRunning = computed(() => {
  return (bot: BotEntity) => task.getRunning(bot.uuid)
})

const leaders = computed(() => {
  return app.getBotList.filter((bot) => {
    const isLeader = bot.leader == ''
    return app.useSubAgent ? isLeader : !isLeader
  })
})

const loadTaskList = async () => {
  try {
    const resp = await request.get('/tasks')
    if (Array.isArray(resp)) {
      task.setHistory(resp as TaskEntity[])
    }
  } catch (err) {
    console.error('load tasks:', err)
  }
}

const setActiveTask = async () => {
  const bot = app.getActive || {}
  if (bot?.uuid && !task.getActive) {
    var active = getRunning.value(bot)
    task.setActive(active?.uuid || '')
    // console.log('active', bot, active)
  } else {
    // wait global data load done
    setTimeout(setActiveTask, 150);
  }
}

onMounted(async () => {
  await loadTaskList()
  await setActiveTask()
})

// Expose methods for parent component access
defineExpose({
  setActiveBot
})
</script>

<template>
  <div id="nav-header">
    <img src="/images/icon.svg">
  </div>
  <div id="nav-container">
    <dl class="nav-list">
      <template v-if="leaders.length < 0">
        <dt class="empty-result">
          {{ $t('common.empty') }}
        </dt>
      </template>
      <template v-for="leader in leaders" :key="leader.uuid">
        <dd :class="clazz(leader)" @click="onSwitchBot(leader)">
          <div class="bot-container">
            <tippy :theme="app.getTheme" placement="left-start" :content="leader.name">
              {{ emoji.get(leader.emoji || 'man_technologist') }}
            </tippy>
            <div v-if="getRunning(leader)" class="running-task">
              <tippy :theme="app.getTheme" placement="left-start" 
                :content="`正在运行: ${getRunning(leader)?.name}`">
                <div class="task-indicator">●</div>
              </tippy>
            </div>
          </div>
        </dd>
      </template>
      <dt class="nav-footer">
        <ProfileSet />
        <SwitchSet />
      </dt>
    </dl>
  </div>
</template>

<style scoped>
#nav-header {
  width: 100%;
  display: flex;
  align-items: center;
  flex-direction: row;
  justify-content: center;
  height: var(--nav-height);
  box-sizing: border-box;
  border-bottom: 1px solid var(--color-divider);
}

#nav-header>img {
  width: 30px;
  margin-left: 2px;
}

#nav-container {
  flex-grow: 1;
  width: inherit;
  margin: 0px auto;
  position: relative;
}

.nav-list {
  margin: -2px 0px 0 0px;
  padding-inline-start: 0px;
}

.nav-list>dd {
  cursor: pointer;
  margin: 12px auto;
  font-size: 25px;
  width: 32px;
  height: 32px;
  line-height: 32px;
  border-radius: 5px;
  text-align: center;
}

.nav-list>dd.active {
  /* background-color: var(--color-divider); */
  box-shadow: 2px 3px 6px var(--color-tertiary);
}

.nav-footer {
  display: flex;
  gap: 10px;
  left: 0px;
  bottom: 10px;
  padding: 0 10px;
  position: absolute;
  align-items: center;
  flex-direction: column;
  width: -webkit-fill-available;
}

.bot-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.running-task {
  display: flex;
  bottom: -2px;
  right: -2px;
  position: absolute;
}

.task-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  font-size: 8px;
  line-height: 8px;
  color: #10b981;
  background-color: #10b981;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% {
    opacity: 1;
    transform: scale(1);
  }

  50% {
    opacity: 0.75;
    filter: blur(1px);
    transform: scale(1.2);
  }

  100% {
    opacity: 1;
    transform: scale(1);
  }
}
</style>
