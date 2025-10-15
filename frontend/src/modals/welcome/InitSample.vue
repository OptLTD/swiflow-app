<script setup lang="ts">
const emit = defineEmits<{
  selectTask: [task: any]
  submitTask: [task: any]
}>()

// Sample tasks configuration with enhanced data structure
const sampleTasks = [
  {
    title: '制作工资单',
    botKey: 'office-bot',
    botName: 'Office Bot',
    brief: '请帮我用Excel制作一份工资单',
    prompt: '请帮我用Excel制作一份工资单，包括员工姓名、工资、扣款项、扣款金额、扣款日期等, 每个员工的工资单之间用空行隔开。'
  },
  {
    title: '单据识别',
    botKey: 'office-bot',
    botName: 'Office Bot',
    brief: '请帮我把单据整理成Excel表格',
    prompt: '请帮我识别我的单据，包括工资单、发票、订单等，并将结果以Excel表格的形式返回。'
  },
  {
    title: '数据分析报告',
    botKey: 'office-bot',
    botName: 'DataAnalyst Bot',
    brief: '从Excel文件生成数据分析报告',
    prompt: '请分析我提供的Excel数据文件，生成包含统计图表、趋势分析和洞察结论的完整数据报告。'
  },
  {
    title: '更多玩法',
    botKey: 'office-bot',
    botName: 'Office Bot',
    brief: '更多玩法敬请期待',
    prompt: '你都会点什么？'
  }
]

// Task selection handler - now uses task title as identifier
const handleSelect = (task: any) => {
  emit('selectTask', task)
}

// Try task handler - emits the complete task object
const handleSubmit = (task: any) => {
  emit('submitTask', task)
}

// Expose component capabilities
defineExpose({
  handleSelect,
  handleTryTask: handleSubmit,
})
</script>

<template>
  <div class="step-content">
    <h3>{{ $t('welcome.selectSampleTask') }}</h3>
    <p class="step-description">{{ $t('welcome.selectSampleTaskDesc') }}</p>
    <div class="tasks-grid">
      <div v-for="(task, idx) in sampleTasks" :key="idx" 
        class="task-card"  @click="handleSelect(task)">
        <div class="task-header">
          <h4 class="task-title">{{ task.title }}</h4>
        </div>

        <p class="task-brief">{{ task.brief }}</p>

        <div class="task-footer">
          <div class="bot-info">
            <span class="bot-icon">🤖</span>
            <span class="bot-name">{{ task.botName }}</span>
          </div>
          <button class="try-button" @click.stop="handleSubmit(task)">
            {{ $t('welcome.tryButton') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.step-content {
  text-align: center;
}

.step-content h3 {
  color: var(--color-primary);
  margin-bottom: 10px;
}

.step-description {
  color: var(--color-text-secondary);
}

.tasks-grid {
  display: grid;
  gap: 10px;
  margin: 0 auto;
  grid-template-columns: 1fr 1fr;
}

.task-card {
  text-align: left;
  display: flex;
  flex-direction: column;
  border-radius: 5px;
  padding: 12px 15px;
  cursor: pointer;
  position: relative;
  transition: all 0.3s ease;
  background: var(--bg-light);
  border: 1px solid var(--color-divider);
}

.task-card:hover {
  background: var(--bg-light);
  border-color: var(--color-tertiary);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.task-header {
  /* margin-bottom: 5px; */
}


.task-title {
  font-weight: 600;
  font-size: 16px;
  color: var(--text-main);
  margin: 0 auto;
}

.task-brief {
  margin: 5px auto;
  min-height: 4rem;
  line-height: 1.5;
  text-align: left;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  text-overflow: ellipsis;
  width: -webkit-fill-available;
  color: var(--color-text-secondary);
}

.task-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
}



.bot-info {
  display: flex;
  align-items: center;
  gap: 6px;
}

.bot-icon {
  font-size: 16px;
}

.bot-name {
  color: var(--color-primary);
  font-weight: 500;
  font-size: 13px;
}

.try-button {
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  flex-shrink: 0;
  padding: 6px 12px;
  border-radius: 6px;
  transition: all 0.2s ease;
  border: 1px solid var(--color-tertiary);
}

.try-button:hover {
  color: var(--bg-main);
  transform: translateY(-1px);
  background-color: var(--color-primary);
}

.try-button:active {
  transform: translateY(0);
}

/* Dark theme enhancements */
@media (prefers-color-scheme: dark) {
  .task-card:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }
  
  .tag {
    background: var(--color-primary-dark);
  }
}

/* Responsive design for smaller screens */
@media (max-width: 640px) {
  .tasks-grid {
    max-width: 400px;
    grid-template-columns: 1fr;
  }
}
</style>