<script setup lang="ts">
import { request, alert } from '@/support/index';
import { onMounted, ref, unref } from 'vue';
import { VueFinalModal } from 'vue-final-modal'
import FormBrowser from '@/widgets/FormBrowser.vue'

onMounted(async () => {
  await doLoad()
})

const config = ref<CfgBrowserMeta>()
const theForm = ref<typeof FormBrowser>()
const emit = defineEmits(['submit', 'cancel'])

const doLoad = async () => {
  try {
    const url = `/toolenv?act=browser-info`
    const resp = await request.post(url, {})
    config.value = (resp || {}) as CfgBrowserMeta
  } catch (err) {
    console.error('test-db-config:', err)
  }
}

const doSumbit = async () => {
  const data = unref(theForm)!.getFormModel()
  try {
    const url = `/toolenv?act=save-browser`
    const resp = await request.post(url, data)
    alert((resp as any)?.errmsg || 'success')
  } catch (err) {
    console.error('set browser:', err)
  } finally {
    emit('submit')
  }
}

</script>

<template>
  <VueFinalModal
    modalId="theBrowserModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <h2> 浏览器配置 </h2>
    <div class="door-box">
      <dl>
        <dt>这是什么，它能干什么？</dt>
        <dd>这是基于 Chrome、Edge 浏览器 Headless 提供的功能，简单来说就是 Swiflow 启动了一个你看不见的浏览器。</dd>
        <dd>人工智能能通过此浏览器来访问互联网，并代替你操作网页，免去了你自己的各种繁琐操作</dd>
        <dt>它安全么，会窃取我的浏览记录和账号密码么？</dt>
        <dd>它很安全，和您自己的浏览器数据是完全隔离，您不用担心自己的账号和密码泄露</dd>
        <dt>它能帮我完成所有基于浏览器的自动化操作么？</dt>
        <dd>目前 Swiflow 还只能控制浏览器搜索和访问网页</dd>
        <dd>更多能力敬请期待～</dd>
      </dl>
      <FormBrowser :config="config" ref="theForm"/>
    </div>
    <div class="actions">
      <button class="btn-submit" @click="doSumbit">
        {{ $t('common.save') }}
      </button>
      <button class="btn-cancel" @click="emit('cancel')">
        {{ $t('common.cancel') }}
      </button>
    </div>
  </VueFinalModal>
</template>

<style>
@import url('@/styles/modal.css');
</style>