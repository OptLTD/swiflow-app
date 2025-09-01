<script setup lang="ts">
import { onMounted, ref, unref } from 'vue'
import { VueFinalModal } from 'vue-final-modal'
import FormModel from '@/widgets/FormModel.vue';
import { request, alert } from '@/support/index';

const props = defineProps({
  from: {
    type: String,
    default: ''
  },
  provider: {
    type: String,
    default: ''
  },
  gateway: {
    type: String,
    default: ''
  },
})

// 添加选项卡状态
const currTab = ref('apikey') 

const errmsg = ref<string>('')
const config = ref<ModelMeta>()
const models = ref<ModelResp>({})
const theForm = ref<typeof FormModel>()
const emit = defineEmits(['submit', 'cancel'])
const doLoad = async (name: string) => {
  try {
    const url = `/setting?act=get-model`
    const resp = await request.get(url) as any
    models.value = resp.models || {}
    if (resp && resp.useModel) {
      config.value = resp.useModel as ModelMeta
    }
    if (props.from == 'provider' && models.value[name]) {
      config.value = models.value[props.from] as ModelMeta
    }
  } catch (err) {
  } finally {
    if (!config.value || !config.value.provider) {
      config.value = {provider: 'doubao'} as ModelMeta
    }
  }
}

const doSubmit = async () => {
  const data = unref(theForm)!.getFormModel()
  if (!data) {
    errmsg.value =  'invalid data'
    return
  }
  if (props.from == 'provider') {
    doSaveProvider(data)
  } else {
    doSaveUseModel(data)
  }
}

const doSaveUseModel = async (data: any) => {
  try {
    const url = `/setting?act=set-model`
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

const doSaveProvider = async (data: any) => {
  try {
    const url = `/setting?act=set-provider`
    const resp = await request.post(url, data)
    errmsg.value = (resp as any)?.errmsg || 'success'
  } catch (err) {
    errmsg.value = err as string
  } finally {
    if (errmsg.value=='success') {
      emit('submit', data)
      alert('SUCCESS')
    }
  }
}

onMounted(async () => {
  await doLoad(props.provider)
})

const gotoSignUp = async () => {
  const path = 'authorization?from=swiflow-app'
  const signup = document.getElementById('signupUrl')
  signup?.setAttribute('href', `${props.gateway}/${path}`)
  return signup && signup.click && signup.click()
}
</script>

<template>
  <VueFinalModal modalId="theModelProvider" class="swiflow-modal-wrapper" content-class="modal-content"
    overlay-transition="vfm-fade" content-transition="vfm-fade">
    <h2 class="modal-title">{{ $t('menu.modelSet') }}</h2>

    <div class="door-box">
      <img src="/images/art-llm.png" class="art-image">

      <!-- 根据选项卡显示不同内容 -->
      <div class="form-content">

        <!-- 添加选项卡 -->
        <div class="tab-container">
          <div class="tab-options">
            <label class="tab-option" :class="{ active: currTab === 'apikey' }">
              <input type="radio" v-model="currTab" value="apikey" class="tab-radio" />
              <span class="tab-label">✓ 我有Api Key</span>
            </label>
            <label class="tab-option" :class="{ active: currTab === 'trial' }">
              <input type="radio" v-model="currTab" value="trial" class="tab-radio" />
              <span class="tab-label">✓ 注册体验</span>
            </label>
          </div>
        </div>
        <div v-if="currTab === 'trial'" class="trial-content">
          <div class="trial-info">
            <p>当前模式由我们的认证服务商 Swiflow 提供能力支持</p>
            <p>注册成功后，您可以免费体验由 Swiflow 提供的 AI 服务</p>
            <div class="trial-features">
              <div class="feature-item">
                <span class="feature-icon">🚀</span>
                <span>快速开始，无需配置</span>
              </div>
              <div class="feature-item">
                <span class="feature-icon">💡</span>
                <span>体验完整功能</span>
              </div>
              <div class="feature-item">
                <span class="feature-icon">🔒</span>
                <span>数据安全保护</span>
              </div>
            </div>
          </div>
        </div>
        <FormModel v-else :config="config" ref="theForm" :models="models" />
      </div>
    </div>
    <div class="actions">
      <button class="btn-submit" @click="gotoSignUp" v-if="currTab === 'trial'">
        {{ $t('common.gotoSignUp') }}
        <a target="_blank" id="signupUrl" />
      </button>
      <button class="btn-submit" @click="doSubmit" v-else-if="currTab === 'apikey'">
        {{ $t('common.save') }}
      </button>
      <button class="btn-cancel" @click="emit('cancel')">
        {{ $t('common.cancel') }}
      </button>
    </div>
  </VueFinalModal>
</template>

<style scoped>
@import url('@/styles/modal.css');
:global(.modal-content){
  min-width: 680px!important;
  max-width: 680px!important;
}

/* 选项卡样式 */
.tab-container {
  margin-bottom: 20px;
}

.tab-options {
  display: flex;
  gap: 0;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #e1e5e9;
}

.tab-option {
  flex: 1;
  position: relative;
  cursor: pointer;
  transition: all 0.2s ease;
}

.tab-radio {
  display: none;
}

.tab-label {
  display: block;
  padding: 12px 16px;
  text-align: center;
  background-color: #f8f9fa;
  color: #6c757d;
  font-weight: 500;
  border-right: 1px solid #e1e5e9;
  transition: all 0.2s ease;
}

.tab-option:last-child .tab-label {
  border-right: none;
}

.tab-option.active .tab-label {
  background-color: #007bff;
  color: white;
}

.tab-option:hover:not(.active) .tab-label {
  background-color: #e9ecef;
  color: #495057;
}

/* 体验模式样式 */
.trial-content {
  display: flex;
  width: 100%;
  min-height: 275px;
}

.trial-info {
  width: 100%;
  max-width: 400px; 
  text-align: center;
}

.trial-info h3 {
  color: #333;
  margin-bottom: 16px;
  font-size: 1.5rem;
}

.trial-info p {
  color: #6c757d;
  line-height: 1.2;
  margin: 1rem 0;
}
.trial-info p:last-of-type{
  margin-bottom: 1.75rem;
}

.trial-features {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background-color: #f8f9fa;
  border-radius: 8px;
  border-left: 4px solid #007bff;
}

.feature-icon {
  font-size: 1.2rem;
}

/* 确保表单内容区域宽度一致 */
.form-content {
  width: 100%;
  box-sizing: border-box;
}

@media (max-width: 760px) {
  .art-image {
    display: none;
  }
  :global(.modal-content){
    min-width: var(--fk-max-width-input)!important;
    max-width: var(--fk-max-width-input)!important;
  }
  
  .tab-container {
    padding: 0 10px;
  }
  
  .trial-content {
    padding: 15px;
  }
}
</style>