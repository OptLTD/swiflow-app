<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { alert } from '@/support';
import { VueFinalModal } from 'vue-final-modal';
import { doInstall } from '@/logics/mcp';
import { checkNetEnv } from '@/logics/mcp';
import { checkMcpEnv } from '@/logics/mcp';

const emit = defineEmits(['submit', 'cancel'])

const netEnv = ref('')
const isReady = ref(false)
const loading = ref(false)
const running = ref(false)

const updatePyEnv = (ok: boolean) => {
  isReady.value = ok
  if (ok) {
    alert('已安装成功')
    emit('cancel')
  }
}

onMounted(async () => {
  loading.value = true
  netEnv.value = await checkNetEnv()
  loading.value = false
  checkMcpEnv(updatePyEnv)
})

const doInstallHandler = async () => {
  running.value = true
  await doInstall(netEnv.value, 'py', updatePyEnv)
  running.value = false
}
</script>

<template>
  <VueFinalModal
    modalId="thePythonModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <h2> 初始化 Python 环境 </h2>
    <dl>
      <dt>Python 是什么？</dt>
      <dd>Python 是一种伟大的语言，它支持广泛的应用程序开发，从简单的文字处理到 WWW 浏览器再到游戏。</dd>
      <dd>Python 拥有丰富的标准库和第三方库，被应用到各种项目中间接极大的提高了人们的工作效率。</dd>
      <dt>为什么要在我电脑上安装 Python？</dt>
      <dd>
        AI 的编程能力目前已经是有目共睹，安装 Python 后你便可以通过 Swiflow 召唤 AI 在你电脑上编写 Python 代码并运行，借此来优化你的工作方式提升工作效率。
      </dd>
      <dt>安装 Python 会对我的电脑带来危害么？</dt>
      <dd>安装 Python 不会给你的电脑带来危害，你也可以自己从<a href="https://www.python.org/" target="_blank">官网下载安装包</a>的方式来自行安装 Python，此方式只是根据你的电脑简化了安装过程</dd>
      <dd>AI 在你的电脑上运行代码时 Swiflow 做了环境隔离，以阻止任何未知因素给你的电脑带来的危害</dd>
    </dl>
  
    <div class="actions">
      <button class="btn-submit" :disabled="running" @click="doInstallHandler">
        {{ running ? '安装中...' : '安装' }}
      </button>
      <button class="btn-cancel" :disabled="running" @click="emit('cancel')">
        取消
      </button>
    </div>
  </VueFinalModal>
</template>

<style>
@import url('@/styles/modal.css');
.init-python {
  min-width: 520px;
  min-height: 320px;
}
</style>