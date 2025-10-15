<script setup lang="ts">
</script>

<template>
  <div class="step-content">
    <h3>{{ $t('welcome.introTitle') }}</h3>
    <div class="intro-content display-block">
      <!-- Feature introduction -->
      <div class="feature-grid">
        <div class="feature-item">
          <span class="feature-icon">🤖</span>
          <div class="feature-text">
            <h4>智能AI助手</h4>
            <p>支持多种AI模型，提供强大的对话和任务处理能力</p>
          </div>
        </div>
        <div class="feature-item">
          <span class="feature-icon">🔧</span>
          <div class="feature-text">
            <h4>丰富工具集成</h4>
            <p>集成Python、Node.js等开发工具，支持MCP协议扩展</p>
          </div>
        </div>
        <div class="feature-item">
          <span class="feature-icon">⚡</span>
          <div class="feature-text">
            <h4>自动化工作流</h4>
            <p>创建和执行复杂的自动化任务，提升工作效率</p>
          </div>
        </div>
        <div class="feature-item">
          <span class="feature-icon">📊</span>
          <div class="feature-text">
            <h4>数据分析处理</h4>
            <p>强大的数据处理和分析能力，支持多种数据格式</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.step-content {
  text-align: center;
  color: var(--text-main);
}

.intro-content {
  margin-top: 20px;
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
  margin-top: 30px;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 15px;
  padding: 20px;
  border: 1px solid var(--color-tertiary);
  border-radius: 8px;
  color: var(--text-main);
  background: var(--bg-light);
  transition: all 0.3s ease;
}

.feature-item:hover {
  transform: translateY(-2px);
  background: var(--color-bg-hover);
  border-color: var(--color-primary);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

[data-theme="dark"] .feature-item:hover {
  box-shadow: 0 4px 12px rgba(255, 255, 255, 0.1);
}

.feature-icon {
  font-size: 24px;
  flex-shrink: 0;
}

.feature-text {
  text-align: left;
}

.feature-text h4 {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-primary);
}

.feature-text p {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-secondary);
  line-height: 1.4;
}
</style>