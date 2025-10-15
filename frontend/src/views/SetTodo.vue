<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { toast } from 'vue3-toastify';
import { computed, onMounted, ref } from 'vue';
import { confirm, request, md } from '@/support';
import BasicMenu from './widgets/BasicMenu.vue';
import SetHeader from './widgets/SetHeader.vue';
import SwitchInput from '@/widgets/SwitchInput.vue'
import { useTaskStore } from '@/stores/task';


const task = useTaskStore()
const { t } = useI18n({
  inheritLocale: true,
  useScope: 'global'
})
const items = ref<TodoEntity[]>()
const active = ref('' as string)
const currTab = ref('' as string)
const current = ref({} as TodoEntity)

onMounted(async () => {
  await doLoad('todo')
})

const onSelect = (item: TodoEntity) => {
  active.value = String(item.uuid)
  current.value = item
}

const titles = computed(() => {
  const result = {} as any
  task.getHistory.forEach((item) => {
    result[item.uuid] = item.name
  })
  return result
})

const doLoad = async (type: string) => {
  if (currTab.value == type) {
    return
  }
  try {
    currTab.value = type;
    const url = `/todo?act=get-${type}`
    const resp = await request.post(url) 
    items.value = resp as TodoEntity[]
    if (items.value?.length > 0) {
      onSelect(items.value[0])
      console.log('items', titles.value)
      items.value.forEach((item) => {
        item.task = titles.value[item.task]
        item.todo = item.todo.substring(0, 22)
      })
    }
  } catch (err) {
    console.error('get todo:', err)
    return toast('ERROR:'+err)
  }
}

const setDone = async (item: TodoEntity) => {
  if (item.done) {
    return
  }
  const msg = t('tips.setDoneMsg')
  // const tip = t('tips.setDoneTip')
  const answer = await confirm(msg);
  if (!answer) {
    return
  }
  try {
    const url = `/todo?act=set-done&uuid=${item.uuid}`
    const resp = await request.post(url) 
    if (!resp || (resp as any)['errmsg']) {
      return toast((resp as any)['errmsg'])
    }
    items.value = items.value?.filter(x => {
      return x.uuid != item.uuid
    })
  } catch (err) {
    console.error('get mcp:', err)
    return toast('ERROR:'+err)
  }
}

</script>

<template>
  <SetHeader :title="$t('menu.todoSet')"/>
  <div id="todo-setting" class="set-view">
    <div id="todo-menu" class="set-menu">
      <div class="todo-tabs">
        <button class="todo-tab" @click="doLoad('todo')"
          :class="{active: currTab == 'todo'}">
          TODO
        </button>
        <button class="todo-tab" @click="doLoad('done')"
          :class="{active: currTab == 'done'}">
          DONE
        </button>
      </div>
      <BasicMenu
        :items="items"
        :keyby="'uuid'"
        :active="active"
        @click="onSelect">
        <template v-slot="{ item }">
          <div class="item-header">
            <h5>{{ item.todo }}</h5>
            <p>{{ item.task }}</p>
          </div>
          <div class="item-action">
            <span class="btn-detail">
              {{ item.time }}
            </span>
            <SwitchInput v-if="!item.done"
              :id="'switch-' + item.name"
              :model-value="item.done+1"
              @change="setDone(item)"
            />
            <SwitchInput v-if="item.done"
              :id="'switch-' + item.name"
              :model-value="false"
              :disabled="true"
            />
          </div>
        </template>
      </BasicMenu>
    </div>
    <div id="todo-panel" class="set-main">
      <div v-html="md.render(current.todo || '')"/>
    </div>
  </div>
</template>

<style scoped>
@import url('@/styles/setting.css');

.todo-tabs{
  display: flex;
  margin-bottom: 10px;
  justify-content: space-around;
}
.todo-tabs>.todo-tab {
  flex: 1 1 0;
  border-width: 0px;
}
.todo-tabs>.active{
  border-width: 2px;
}

#todo-panel {
  padding: 0 15px;
}
</style>
