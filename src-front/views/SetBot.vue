<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { isEmpty } from 'lodash-es';
import { toast } from 'vue3-toastify';
import { onMounted, watch, ref, computed } from 'vue';
import { useAppStore } from '@/stores/app';
import GroupMenu from './widgets/GroupMenu.vue';
import SetHeader from './widgets/SetHeader.vue';
import FormSetBot from '@/widgets/FormSetBot.vue'
import { confirm, request, alert } from '@/support';
import { showInputModal } from '@/logics/popup';

const app = useAppStore()
const active = ref('' as string)
const current = ref({} as BotEntity)
const theForm = ref<typeof FormSetBot>()

const groupedBots = computed(() => {
  const leaders: BotEntity[] = []
  const workers: BotEntity[] = []
  
  app.getBotList.forEach(bot => {
    if (!bot.leader) {
      leaders.push(bot)
      workers.push({
        ...bot, leader: bot.uuid,
        desc: 'this is bot leader',
      })
    }
  })
  app.getBotList.forEach(bot => {
    if (bot.leader) {
      workers.push(bot)
    }
  })
  return { leaders, workers }
})

const botLeaders = computed(() => {
  return groupedBots.value.leaders.map(bot => ({
    label: bot.name, value: bot.uuid,
    group: 'leader', other: bot
  } as MenuMeta))
})

const botWorkers = computed(() => {
  return groupedBots.value.workers.map(bot => ({
    label: bot.name, value: bot.uuid,
    group: bot.leader, other: bot
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
    return app.getBotList.length
  } catch (error) {
    console.error('Error in resetItems:', error)
    return 0
  }
}
const resetBots = (act: string, {uuid, name, leader}: BotEntity) => {
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
      bots.push({uuid, name, leader} as BotEntity)
    }
    app.setBotList(bots.filter(x => x.name))
    if (resetItems() > 0 && act == 'add-bot') {
      active.value = uuid
    }
  } catch (error) {
    console.error('Error in resetBots:', error)
  }
}

const onSelectBot = async (item: MenuMeta) => {
  try {
    await doLoad(item.value)
  } catch (error) {
    console.error('Error in onSelect:', error)
  }
}

const onCreateBot = (leaderId: string) => {
  current.value = {
    'leader': leaderId
  } as BotEntity
  active.value = ''
}

const onRemoveBot = async (item: MenuMeta) => {
  const msg = t('tips.delBotMsg')
  const answer = await confirm(msg)
  if (!answer) return

  await doRemoveBot(item.value)

  if (app.getBotList.length > 0) {
    const curr = app.getBotList[0]
    active.value = curr.uuid
    await doLoad(curr.uuid)
    return
  }

  // Reset bots if no bots left
  if (await confirm(t('tips.initBotMsg'))) {
    await doInit()
  }
}

const doRemoveBot = async (uuid: string) => {
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
    current.value = { uuid: uuid, name: '', provider: '' } as BotEntity
  }
}
const doSaveBot = () => {
  try {
    const form = theForm.value
    const formData = form?.getFormModel()
    if (formData && formData.name) {
      delete(formData.sysPrompt)
      doSubmit(formData)
    }
  } catch (error) {
    console.error('Error in doSaveBot:', error)
  }
}

const doSubmit = async (bot: BotEntity) => {
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

const onEditBot = (item: MenuMeta) => {
  // Show modal to edit bot description
  const bot = item.other as BotEntity
  const props = {
    input: bot.desc, 
    tips: t('common.editDescTips'), 
    title: t('common.editDescTitle'), 
   }
  showInputModal(props, async (text: string) => {
    bot.desc = text
    await doUpdateBotDesc(bot)
    if (active.value === item.value) {
      await doLoad(item.value)
    }
  })
}

const doUpdateBotDesc = async (bot: BotEntity) => {
  try {
    const url = `/bot?act=set-bot&uuid=${bot.uuid}`
    const resp = await request.post<any>(url, { uuid: bot.uuid, desc: bot.desc })
    if (resp?.errmsg) {
      alert(resp.errmsg)
      return
    }
    // Update the bot in the store
    const botIndex = app.getBotList.findIndex(b => b.uuid === bot.uuid)
    if (botIndex !== -1) {
      app.getBotList[botIndex].desc = bot.desc
    }
    toast('SUCCESS')
  } catch (err) {
    console.error('update bot desc:', err)
    toast('ERROR:' + err)
  }
}

</script>

<template>
  <SetHeader :title="$t('menu.botSet')"/>
  <div id="bot-setting" class="set-view">
    <div id="bot-menu" class="set-menu">
      <GroupMenu
        :divide="false"
        :current="active"
        :items="botWorkers"
        :group="botLeaders"
        :active="botActive"
        @click="onSelectBot"
        @create="onCreateBot"
        @remove="onRemoveBot">
        <template v-slot="{ item }">
          <div class="item-header">
            <h5>{{ item.label }}</h5>
            <p v-if="(item.other as BotEntity)?.desc">
              {{ (item.other as BotEntity).desc }}
            </p>
            <p v-else>
              {{ $t('common.botNoDesc') }}
            </p>
          </div>
          <div class="item-action">
            <button @click.stop="onRemoveBot(item)" 
              class="btn-icon icon-small btn-remove" 
            />
            <a class="btn-modify" 
              @click="onEditBot(item)">
              {{ $t('common.edit') }}
            </a>
          </div>
        </template>
      </GroupMenu>
      <button class="btn-add-new" @click="onCreateBot('')">
        {{ $t('common.addBot') }}
      </button>
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

.item-action .btn-modify{
  margin-top: 5px;
  margin-bottom: 0;
}
</style>
