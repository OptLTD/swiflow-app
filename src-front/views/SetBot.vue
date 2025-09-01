<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { isEmpty } from 'lodash-es';
import { toast } from 'vue3-toastify';
import { onMounted, watch, ref } from 'vue';
import { useAppStore } from '@/stores/app';
import BasicMenu from './widgets/BasicMenu.vue';
import SetHeader from './widgets/SetHeader.vue';
import FormSetBot from '@/widgets/FormSetBot.vue'
import { confirm, request, alert } from '@/support';

const app = useAppStore()
const items = ref<MenuMeta[]>([])
const active = ref('' as string)
const current = ref({} as BotEntity)
const theForm = ref<typeof FormSetBot>()

const { t } = useI18n({
  inheritLocale: true,
  useScope: 'global'
})

watch(() => app.getAction, (val) => {
  try {
    if (val == 'default') {
      return
    }
    return resetItems()
  } catch (error) {
    console.error('Error in watch:', error)
  }
})
onMounted(async () => {
  try {
    if (resetItems() > 0) {
      const curr = app.getBotList[0]
      if (curr && curr.uuid) {
        await doLoad(curr.uuid)
        active.value = curr.uuid
      }
    } else {
      // 重置bots
      if (await confirm(t('tips.initBotMsg'))){
        await doInit()
      }
    }
  } catch (error) {
    console.error('Error in onMounted:', error)
  }
})

const resetItems = () => {
  try {
    items.value = app.getBotList.map((item) => {
      return {
        label: item.name || '',
        value: item.uuid || '',
      }
    })
    return app.getBotList.length
  } catch (error) {
    console.error('Error in resetItems:', error)
    items.value = []
    return 0
  }
}
const resetBots = (act: string, {uuid, name}: BotEntity) => {
  try {
    const bots = app.getBotList
    for(var i in bots) {
      if (bots[i].uuid == uuid) {
        bots[i].name = name
        if (act == 'del-bot') {
          bots[i].name = ''
        }
      }
    }
    if (act == 'add-bot') {
      bots.push({uuid, name} as BotEntity)
    }
    app.setBotList(bots.filter(x => x.name))
    if (resetItems() > 0 && act == 'add-bot') {
      active.value = uuid
    }
  } catch (error) {
    console.error('Error in resetBots:', error)
  }
}

const onCreate = () => {
  try {
    current.value = {'provider': 'doubao'} as BotEntity
  } catch (error) {
    console.error('Error in onCreate:', error)
  }
}

const onRemove = async (item: MenuMeta) => {
  const msg = t('tips.delBotMsg')
  // const tip = t('tips.delBotTip')
  const answer = await confirm(msg)
  if (!answer) return
  await doRemove(item.value)

  if (app.getBotList.length > 0) {
    const curr = app.getBotList[0]
    active.value = curr.uuid
    await doLoad(curr.uuid)
    return
  }

  // 重置bots
  if (await confirm(t('tips.initBotMsg'))){
    await doInit()
  }
}

const onSelect = async (item: MenuMeta) => {
  try {
    await doLoad(item.value)
  } catch (error) {
    console.error('Error in onSelect:', error)
  }
}

const doRemove = async (uuid: string) => {
  try {
    const url = `/bot?act=del-bot&uuid=${uuid}`
    const resp = await request.post(url) 
    if (!isEmpty(resp)) {
      resetBots('del-bot', resp as BotEntity)
    }
    toast('SUCCESS')
  } catch (err) {
    console.error('get bot:', err)
    toast('ERROR:'+err)
  }
}

const doInit = async () => {
  try {
    const url = `/bot?act=init-bot`
    const resp = await request.post(url)
    app.setBotList(resp as BotEntity[])
    if (resetItems() > 0) {
      const curr = app.getBotList[0]
      if (curr && curr.uuid) {
        app.setActive(curr)
        await doLoad(curr.uuid)
        active.value = curr.uuid
      }
    }
  } catch (err) {
    console.error('get bot:', err)
    toast('ERROR:' + err)
  }
}

const doLoad = async (uuid: string) => {
  try {
    const url = `/bot?act=get-bot&uuid=${uuid}`
    const resp = await request.post(url) 
    current.value = resp as BotEntity
  } catch (err) {
    console.error('get bot:', err)
    toast('ERROR:'+err)
    // 设置一个默认的bot实体以防止UI错误
    current.value = { uuid: uuid, name: '', provider: 'deepseek' } as BotEntity
  }
}
const doSaveBot = () => {
  try {
    const form = theForm.value
    const formData = form?.getFormModel()
    if (formData && formData.name) {
      delete(formData.sysPrompt)
      doSumbit(formData)
    }
  } catch (error) {
    console.error('Error in doSaveBot:', error)
  }
}

const doSumbit = async (bot: BotEntity) => {
  try {
    const url = `/bot?act=set-bot&uuid=${bot.uuid || ''}`
    const resp = await request.post<any>(url, bot)
    if (resp?.errmsg) {
      alert(resp.errmsg)
      return
    }
    if (!isEmpty(resp) && !bot.uuid) {
      resetBots('add-bot', resp)
    }
    if (!isEmpty(resp) && bot.uuid) {
      resetBots('set-bot', bot)
    }
    console.log('set bot', resp)
    alert('SUCCESS')
  } catch (err) {
    console.error('set bot:', err)
    alert('ERROR:'+err)
  }
}

</script>

<template>
  <SetHeader :title="$t('menu.botSet')"/>
  <div id="bot-setting" class="set-view">
    <div id="bot-menu" class="set-menu">
      <button class="btn-add-new" @click="onCreate">
        {{ $t('common.addBot') }}
      </button>
      <BasicMenu
        :items="items || []"
        :active="active"
        :keyby="'value'"
        @click="onSelect">
        <template v-slot="{ item }">
          <span>{{ item?.label || '' }}</span>
          <a class="btn-icon btn-remove" 
            @click.stop="onRemove(item)"
          />
        </template>
      </BasicMenu>
    </div>
    <div id="bot-panel" class="set-main">
      <FormSetBot 
        :servers="app.getMcpList"
        :is-multi="app.getIsMulti"
        ref="theForm" :model="current"
        from="setting" @submit="doSaveBot"
      />
    </div>
  </div>
</template>

<style scoped>
@import url('@/styles/setting.css');
#bot-panel{
  padding: 20px 0;
}
#bot-menu .btn-add-new{
  margin: 5px auto;
  font-weight: normal;
  width: -webkit-fill-available;
}
#bot-menu :deep(li) {
  min-height: 24px;
}
#bot-menu .btn-remove{
  display: none;
  margin-right: -5px;
}
#bot-menu .active:hover .btn-remove{
  display: inline-flex;
}
</style>
