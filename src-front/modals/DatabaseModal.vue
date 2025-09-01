<script setup lang="ts">
import { request, alert } from '@/support/index';
import { onMounted, ref, unref } from 'vue'
import { VueFinalModal } from 'vue-final-modal'
import FormMySQL from '@/widgets/FormMySQL.vue';

onMounted(async () => {
  await doLoad()
})

const errmsg = ref<string>('')
const config = ref<CfgMySQLMeta>()
const theForm = ref<typeof FormMySQL>()
const emit = defineEmits(['submit', 'cancel'])
const doLoad = async () => {
  try {
    const url = `/toolenv?act=db-info`
    const resp = await request.get(url)
    config.value = resp as CfgMySQLMeta
  } catch (err) {
    config.value = {} as CfgMySQLMeta
  }
}

const doTest = async () => {
  const data = unref(theForm)!.getFormModel()
  if (!data) {
    errmsg.value =  'invalid data'
    return
  }
  try {
    const url = `/toolenv?act=test-db`
    const resp = await request.post(url, data)
    errmsg.value = (resp as any)?.errmsg || 'success'
  } catch (err) {
    errmsg.value = err as string
    console.error('test-db-config:', err)
  }
}

const doSave = async () => {
  const data = unref(theForm)!.getFormModel()
  if (!data) {
    errmsg.value =  'invalid data'
    return
  }
  try {
    const url = `/toolenv?act=save-db`
    const resp = await request.post(url, data)
    errmsg.value = (resp as any)?.errmsg || 'success'
  } catch (err) {
    errmsg.value = err as string
  } finally {
    if (errmsg.value=='success') {
      alert('SUCCESS')
      emit('submit')
    }
  }
}

</script>

<template>
  <VueFinalModal
    modalId="theDatabaseModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <h2> 数据库配置 </h2>
    <div class="door-box">
      <dl>
        <dt>为什么要配置数据库？</dt>
        <dd>通过 AI 自动操作数据库，我们仅需描述需求便可直接获取结果，极大的提升了效率。</dd>
        <dt>配置完数据库就可以直接使用了么？</dt>
        <dd>理论上是，但在 Bot 提示词中增加常用SQL 使用说明以及数据字典信息，会提升 Bot 的执行效果</dd>
        <dt>我的数据会上传给 AI 么？</dt>
        <dd>是的，通常业务中的需求要通过 N 个 SQL 链式查询后才能得到结果，AI 要通过执行结果决定下一步操作。</dd>
        <dd>如果您关注隐私，可以把不想 AI 查询的数据表放到 Deny List 里，Swiflow 会自动屏蔽这些表的查询。</dd>
      </dl>
      <FormMySQL :mysql="config" ref="theForm" :errmsg="errmsg"/>
    </div>
    <div class="actions">
      <button class="btn-submit" @click="doSave">
        保存连接
      </button>
      <button class="btn-default" @click="doTest">
        测试连接
      </button>
      <button class="btn-cancel" @click="emit('cancel')">
        取消
      </button>
    </div>
  </VueFinalModal>
</template>

<style>
@import url('@/styles/modal.css');
</style>