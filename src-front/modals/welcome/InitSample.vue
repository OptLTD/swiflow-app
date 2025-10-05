<script setup lang="ts">
const emit = defineEmits<{
  selectTask: [task: any]
  submitTask: [task: any]
}>()

// Sample tasks configuration with enhanced data structure
const sampleTasks = [
  {
    title: '网页数据抓取',
    botKey: 'office-bot',
    botName: 'Office Bot',
    brief: '抓取指定网站的数据并进行分析',
    prompt: '请帮我抓取指定网站的数据，包括商品信息、价格、评论等，并进行数据清洗和分析处理。'
  },
  {
    title: '代码审查助手',
    botKey: 'office-bot',
    botName: 'CodeReview Bot',
    brief: '请对我的代码进行全面审查，检查代码规范、性能问题、安全漏洞，并提供具体的改进建议。',
    prompt: '请对我的代码进行全面审查，检查代码规范、性能问题、安全漏洞，并提供具体的改进建议。'
  },
  {
    title: '数据分析报告',
    botKey: 'office-bot',
    botName: 'DataAnalyst Bot',
    brief: '从CSV文件生成数据分析报告',
    prompt: '请分析我提供的CSV数据文件，生成包含统计图表、趋势分析和洞察结论的完整数据报告。'
  },
  {
    title: 'API接口测试',
    botKey: 'office-bot',
    botName: 'APITester Bot',
    brief: '请帮我设计和执行API接口的自动化测试，包括功能测试、性能测试和边界条件测试。',
    prompt: '请帮我设计和执行API接口的自动化测试，包括功能测试、性能测试和边界条件测试。'
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