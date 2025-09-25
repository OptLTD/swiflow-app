<script setup lang="ts">
import { toast } from 'vue3-toastify';
import { onMounted, ref, computed } from 'vue';
import { useAppStore } from '@/stores/app';
import { confirm, request } from '@/support';
import GroupMenu from './widgets/GroupMenu.vue';
import SetHeader from './widgets/SetHeader.vue';
import FormSetMem from '@/widgets/FormSetMem.vue'
import { useI18n } from 'vue-i18n';

const { t } = useI18n({
  inheritLocale: true,
  useScope: 'global'
})
const app = useAppStore()
const mems = ref<MemEntity[]>()
const active = ref('' as string)
const current = ref({} as MemEntity)
const memForm = ref<typeof FormSetMem>()

// Convert bots to MenuMeta structure for GroupMenu compatibility
const botGroups = computed(() => {
  return app.getBotList.map(bot => ({
    label: bot.name,
    value: bot.uuid,
    group: 'bot',
    other: bot
  } as MenuMeta))
})

const botActive = computed(() => {
  if (!app.getActive) {
    return undefined
  }
  const bot = app.getActive
  return {
    label: bot.name, value: bot.uuid,
    group: bot.leader, other: bot
  } as MenuMeta
})

// Convert mems to MenuMeta structure for GroupMenu compatibility
const memItems = computed(() => {
  if (!mems.value) return []
  return mems.value.map(mem => ({
    label: mem.subject || 'Untitled',
    value: String(mem.id),
    group: mem.bot || 'other',
    other: mem
  } as MenuMeta))
})


onMounted(async () => {
  await doLoad()
})

const onSelect = (item: MenuMeta) => {
  active.value = item.value
  // Find the original MemEntity from the MenuMeta
  const mem = mems.value?.find(m => String(m.id) === item.value)
  if (mem) {
    current.value = mem
  }
}

const onCreate = (botId: string) => {
  console.log('onCreate called with botId:', botId)
  current.value = {'bot': botId} as MemEntity
  console.log('current.value after onCreate:', current.value)
  active.value = ''
}

const onRemove = async (item: MenuMeta) => {
  const msg = t('tips.delMemMsg')
  // const tip = t('tips.delMemTip')
  const answer = await confirm(msg);
  if (!answer) {
    return
  }
  await doRemove(item.value)
}

const doRemove = async (uuid: string) => {
  try {
    const url = `/mem?act=del-mem&uuid=${uuid}`
    await request.post(url) 
    await doLoad() // 重新加载数据
    return toast('SUCCESS')
  } catch (err) {
    console.error('get bot:', err)
    return toast('ERROR:'+err)
  }
}

const doLoad = async () => {
  try {
    const url = `/mem?act=get-mem`
    const resp = await request.post(url) 
    mems.value = resp as MemEntity[]
  } catch (err) {
    console.error('get-mem:', err)
    return toast('ERROR:'+err)
  }
}

const onSubmit = () => {
  const theForm = memForm.value
  const formData = theForm!.getFormModel()
  if (formData) {
    if (!formData.bot) {
      toast.error(t('tips.noBotSelect'))
      return
    }
    if (!formData.bot.trim()) {
      toast.error(t('tips.noBotSelect'))
      return
    }
    doSumbit(formData)
  }
}

const doSumbit = async (mem: MemEntity) => {
  try {
    // 验证bot字段不能为空
    if (!mem.bot || mem.bot.trim() === '') {
      toast.error(t('tips.noBotSelect'))
      return
    }
    
    let url = `/mem?act=set-mem`
    if (active.value) {
      url += `&uuid=${active.value}`
    }
    const resp = await request.post(url, mem) as MemEntity
    console.log('set mem', resp)
    await doLoad() // 重新加载数据
    return toast.success('SUCCESS')
  } catch (err) {
    console.error('set mem:', err)
    return toast.error('ERROR:'+err)
  }
}
</script>

<template>
  <SetHeader :title="$t('menu.memSet')"/>
  <div id="mem-setting"  class="set-view">
    <div id="mem-menu"  class="set-menu">
      <GroupMenu
        :current="active"
        :items="memItems"
        :group="botGroups"
        :active="botActive"
        @click="onSelect"
        @create="onCreate"
        @remove="onRemove">
      </GroupMenu>
    </div>
    <div id="mem-panel" class="set-main">
      <FormSetMem ref="memForm" :model="current" @submit="onSubmit"/>
    </div>
  </div>
</template>

<style scoped>
@import url('@/styles/setting.css');
.mem-item {
  display: flex;
  width: fit-content;
  flex-direction: column;
}

.mem-type {
  font-size: 12px;
  color: #666;
  font-weight: normal;
  margin-bottom: 2px;
}
</style>
