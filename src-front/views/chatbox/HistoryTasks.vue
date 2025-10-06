<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import * as emoji from 'node-emoji'
import { useAppStore } from '@/stores/app'
import { useTaskStore } from '@/stores/task'
import { confirm, request } from '@/support'

const { t } = useI18n({
  inheritLocale: true,
  useScope: 'global'
})

const app = useAppStore()
const task = useTaskStore()
onMounted(async () => {
  await doLoadTask()
})

// 关闭历史面板
const closeHistory = () => {
  app.toggleHistory()
}

const doLoadTask = async () => {
  try {
    const resp = await request.get('/tasks')
    task.setHistory(resp as TaskEntity[])
  } catch (e) {
    console.error('use bot:', e)
  }
}

// 获取任务对应的bot信息
const getTaskBot = (item: TaskEntity) => {
  if (!item.botid) return null
  return app.getBotList.find(bot => {
    return bot.uuid === item.botid
  })
}

// 获取状态显示文本
const getStatusText = (state: string) => {
  const statusMap: Record<string, string> = {
    'failed': t('status.failed'),
    'running': t('status.running'),
    'waiting': t('status.waiting'),
    'seek-help': t('status.waiting'),
    'canceled': t('status.canceled'),
    'completed': t('status.completed'),
  }
  return statusMap[state] || state
}

// 获取状态样式类
const getStatusClass = (state: string) => {
  const classMap: Record<string, string> = {
    'running': 'running',
    'waiting': 'running',
    'failed': 'failed',
    'canceled': 'failed',
    'seek-help': 'running',
    'completed': 'success',
  }
  return classMap[state] || 'waiting'
}

// 切换任务
const doSwitchTask = (item: TaskEntity | null) => {
  const isChatView = ['browser', 'default'].includes(app.getAction)
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

// 获取任务样式类
const getTaskClass = (item: TaskEntity) => {
  return item.uuid === task.getActive ? 'active' : ''
}

// 格式化任务名称
const formatTaskName = (name: string) => {
  return name || t('common.untitled')
}

// 删除任务
const doRemoveTask = async(uuid: string) => {
  const msg = t('tips.delTaskMsg')
  const answer = await confirm(msg);
  if (!answer) {
    return
  }
  try {
    const url = `/task?act=del-task&uuid=${uuid}`
    const resp = await request.post(url) as any
    console.log('do del-task success', resp, uuid)
    // 删除成功后刷新任务列表
    await doLoadTask()
    // 如果删除的是当前活跃任务，清空活跃状态
    if (task.getActive === uuid) {
      task.setActive('')
    }
  } catch (err) {
    console.error('del-task:', err)
  }
}
</script>

<template>
  <div class="history-overlay" @click="closeHistory">
    <div class="history-panel" @click.stop>
      <div class="history-header">
        <h3>{{ $t('common.historyTasks') }}</h3>
        <button class="btn-icon btn-close" @click="closeHistory">
          ×
        </button>
      </div>
      
      <div class="history-content">
        <div class="history-list">
          <template v-if="task.getHistory.length === 0">
            <div class="empty-result">
              {{ $t('common.empty') }}
            </div>
          </template>
          
          <template v-else>
            <div v-for="item in task.getHistory" 
              :class="['history-item', getTaskClass(item)]"
              :key="item.uuid" @click="doSwitchTask(item)"
            >
              <div class="task-info">
                <div class="task-header">
                  <span class="task-emoji">
                    {{ emoji.get(getTaskBot(item)?.emoji || 'man_technologist') }}
                  </span>
                  <div class="task-name">{{ formatTaskName(item.name) }}</div>
                </div>
                <div class="task-status">
                  <span :class="['status-badge', getStatusClass(item.state)]">
                    {{ getStatusText(item.state) }}
                  </span>
                </div>
              </div>
              <button class="btn-icon btn-remove" 
                @click.stop="doRemoveTask(item.uuid)"
              />
            </div>
          </template>
        </div>
      </div>
      <!-- <div class="history-footer"></div> -->
    </div>
  </div>
</template>

<style scoped>
.history-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  display: flex;
  justify-content: flex-start;
  background-color: rgba(0, 0, 0, 0.5);
}

.history-panel {
  width: 320px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  animation: slideInLeft 0.3s ease-out;
  background-color: var(--bg-main);
  border-left: 1px solid var(--color-divider);
  border-right: 1px solid var(--color-divider);
}

@keyframes slideInLeft {
  from {
    transform: translateX(-100%);
  }
  to {
    transform: translateX(0);
  }
}

.history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 20px;
  box-sizing: border-box;
  height: var(--nav-height);
  background-color: var(--bg-light);
  border-bottom: 1px solid var(--color-divider);
}

.history-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text);
}

.btn-close {
  width: 24px;
  height: 24px;
  border: none;
  background: none;
  font-size: 18px;
  cursor: pointer;
  color: var(--color-text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.btn-close:hover {
  background-color: var(--color-bg-hover);
  color: var(--color-text);
}

.history-content {
  flex: 1;
  padding: 8px;
  overflow-y: auto;
}

.history-list {
  gap: 4px;
  display: flex;
  flex-direction: column;
}

.empty-result {
  text-align: center;
  padding: 40px 20px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.history-item {
  cursor: pointer;
  position: relative;
  padding: 8px 12px;
  border-radius: 8px;
  transition: all 0.2s ease;
  border: 1px solid transparent;
}

.history-item:hover {
  background-color: var(--color-bg-hover);
}

.history-item.active {
  border-color: var(--color-primary);
  background-color: var(--color-primary-bg);
}
.history-item.active:hover {
  padding-right: 32px;
}
.history-item .btn-remove {
  display: none;
  min-width: 24px;
  min-height: 24px;
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
  border: none;
  background: none;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.history-item.active:hover .btn-run,
.history-item.active:hover .btn-stop {
  right: 28px;
  display: block;
}

.history-item.active:hover .btn-remove {
  display: inline-flex;
}

.task-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex: 1;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.task-emoji {
  font-size: 16px;
  line-height: 1;
  flex-shrink: 0;
}

.task-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.task-status {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.status-badge {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 12px;
  font-weight: 500;
  white-space: nowrap;
  display: inline-block;
  min-width: fit-content;
}

.status-badge.running {
  background-color: var(--color-warning-bg);
  color: var(--color-warning);
}

.status-badge.failed {
  background-color: var(--color-error-bg);
  color: var(--color-error);
}

.status-badge.success {
  background-color: var(--color-success-bg);
  color: var(--color-success);
}



.history-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--color-border);
  background-color: var(--color-bg-secondary);
}
</style>