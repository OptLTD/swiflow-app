<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { debounce } from 'lodash-es';
import { toast } from 'vue3-toastify';
import { watch, onMounted, ref } from 'vue';
import { useAppStore } from '@/stores/app';
import { request } from '@/support/index';
import { FormKit } from '@formkit/vue';
import SetHeader from './widgets/SetHeader.vue';

const app = useAppStore()
const counter = ref<number>(0)
const formModel = ref<SetupMeta>({
  useTheme: 'auto',
  useLanguage: 'zh',

  useWorkPath: '',
  useProxyUrl: '',
  authGateway: '',
  ctxMsgSize: "100",
  maxCallTurns: "25",
  useCopyMode: 'source',
  useIsolated: false,
  useSubAgent: false,
  useDebugMode: false,
  useSandbox: false,
  sendNotifyOn: [],
})
const { t, locale } = useI18n({
  inheritLocale: true,
  useScope: 'global'
})

onMounted(async () => {
  await loadSetting()
})

watch(() => formModel.value, debounce(async (data: any) => {
  counter.value += 1
  if (counter.value <= 1) {
    return
  }
  app.setSetup(data)
  locale.value = data.language
  await saveSetting(data)
  // Show toast only after the final change when user stops modifying
  toast.success('Settings saved successfully!')
}, 1000), { deep: true })

const loadSetting = async () => {
  try {
    const url = '/setting?act=get-setup'
    const resp = await request.get(url) as any
    if (resp['errmsg']) {
      return toast.error(resp['errmsg'])
    }
    Object.assign(formModel.value, resp || {})
    console.log('loadSetting', resp)
  } catch (err) {
    console.error('Failed to loadSetting:', err)
  } finally {
    console.log('loading', counter.value)
  }
}

const saveSetting = async (data: any) => {
  try {
    const url = '/setting?act=put-setup'
    const resp = await request.post(url, data) as any
    if (resp['errmsg']) {
      return toast.error(resp['errmsg'])
    }
  } catch (err) {
    console.error('Failed to loadSetting:', err)
  }
}

const langOptions = [
  { label: '中文', value: 'zh' },
  { label: 'English', value: 'en' }
]
const copyOptions = [
  { label: 'Source', value: 'source' },
  { label: 'Display', value: 'display' }
]
const sendNotifyOptions = [
  { label: '完成时', value: 'complete' },
  { label: '错误时', value: 'error' }
]
const appearanceOptions = () => {
  return [
    { label: t('setting.themeAuto'), value: 'auto' },
    { label: t('setting.themeDark'), value: 'dark' },
    { label: t('setting.themeLight'), value: 'light' }
  ]
}

</script>

<template>
  <SetHeader :title="$t('menu.basicSet')" />
  <div id="base-setting" class="set-view">
    <div id="base-panel" ref="basePanel" class="set-main">
      <div class="form-model" id="basic-setting">
        <FormKit type="form" :actions="false">
          <h3>{{ $t('menu.basicSet') }}</h3>

          <FormKit type="radio" name="useTheme" 
            v-model="formModel.useTheme" 
            :options="appearanceOptions()"
            :label="$t('setting.appearance')" 
          />

          <FormKit type="radio" name="useLanguage" 
            :options="langOptions" 
            v-model="formModel.useLanguage"
            :label="$t('setting.language')" 
          />

          <FormKit type="text" 
            name="useWorkPath" label="工作目录" 
            placeholder="请输入 Bot 工作目录"
            v-model="formModel.useWorkPath" 
          />

          <h3>{{ $t('setting.taskSet') }}</h3>
          <FormKit type="number" min="0" max="100" 
            v-model="formModel.ctxMsgSize"
            name="ctxMsgSize" label="上下文消息数" 
            help="调用 LLM 携带历史消息数量" 
          />
          <FormKit type="number" min="0" max="100" 
            name="maxCallTurns" label="最大调用轮次"
            v-model="formModel.maxCallTurns" 
          />

          <FormKit type="checkbox" 
            name="sendNotifyOn" label="当*时发送通知" 
            :options="sendNotifyOptions"
            v-model="formModel.sendNotifyOn" 
          />
          <FormKit type="radio" 
            name="useCopyMode" :label="$t('setting.useCopy')" 
            v-model="formModel.useCopyMode" :options="copyOptions" 
          />

          <h3>{{ $t('setting.otherSet') }}</h3>
          <div class="row">
            <FormKit type="checkbox" 
              v-model="formModel.useIsolated" 
              name="useIsolated" label="独立供应商" 
            />
            <FormKit type="checkbox" 
              v-model="formModel.useSubAgent" 
              name="useSubAgent" label="子智能体" 
            />
            <!-- <FormKit type="checkbox" 
              v-model="formModel.useSandbox" 
              name="useSandbox" label="沙箱模式" 
            /> -->
            <FormKit type="checkbox" 
              v-model="formModel.useDebugMode" 
              name="useDebugMode" label="开发者模式" 
            />
          </div>
          <FormKit type="text" 
            v-model="formModel.authGateway" 
            name="authGateway" label="认证网关"
            placeholder="如 http://auth.swiflow.cc" 
          />
          <FormKit type="text" 
            v-model="formModel.useProxyUrl" 
            name="useProxyUrl" label="代理地址 (PROXY_URL)"
            placeholder="如 http://user:pass@host:port 或 socks5://host:port" 
          />
        </FormKit>
      </div>
      <!-- 其他设置面板可后续补充 -->
    </div>
  </div>
</template>

<style scoped>
@import url('@/styles/form.css');

#base-panel {
  margin: 0 auto;
  max-width: 720px;
}

#basic-setting {
  flex: 1;
  padding: 0 5px;
  margin: 0 auto;
  max-width: 960px;
  overflow-y: auto;
  width: -webkit-fill-available;
  height: calc(100vh - var(--nav-height));
}

.form-model h3 {
  width: 25rem;
  margin: 0 auto;
  margin-top: 25px;
  margin-bottom: 12px;
}

.form-model .row {
  display: flex;
  gap: 1rem;
}

.form-model :deep(.formkit-wrapper),
.form-model :deep(.formkit-fieldset) {
  margin: 0 auto;
}

.form-model>form {
  width: 25rem;
  margin: 0 auto;
}
</style>
