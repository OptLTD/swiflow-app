<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { toast } from 'vue3-toastify';
import { request } from '@/support/index';
import { useAppStore } from '@/stores/app';
import { showSetMcpModal } from '@/logics/popup'
import BasicMenu from './widgets/BasicMenu.vue';
import SetHeader from './widgets/SetHeader.vue';
import McpMarket from './widgets/McpMarket.vue';
import McpToolTest from './widgets/McpToolTest.vue'
import SwitchInput from '@/widgets/SwitchInput.vue'

const app = useAppStore()
const items = ref<McpServer[]>()
const active = ref('' as string)
const current = ref({} as McpServer)
const loading = ref<Record<string, boolean>>({})

onMounted(async () => {
  await doLoad()
})

const onCreate = () => {
  const mcp = { type: 'stdio' } as McpServer
  showSetMcpModal(mcp , async (item: McpServer) => {
    await doLoad()
    const find = items.value?.find(x => {
      return x.name == item?.name
    }) as McpServer
    find && onSelect(find)
  })
}

const onDetail = (mcp: McpServer) => {
  showSetMcpModal(mcp , async (item: McpServer, name: string) => {
    await doLoad()
    if (name == 'delete') {
      active.value = ''
      return
    }
    const find = items.value?.find(x => {
      return x.name == item?.name
    }) as McpServer
    find && onSelect(find)
  })
}

const onSelect = (item: McpServer) => {
  active.value = item.uuid
  current.value = item as McpServer
}

const onSwitch = async (item: McpServer, status: McpStatus) => {
  console.log(status, 'enable')
  if (!status.enable) {
    return await doActive(item.uuid)
  }
  try {
    const url = `/mcp?act=disable&uuid=${item.uuid}`
    const resp = await request.post(url) 
    if (!resp || (resp as any)['errmsg']) {
      return toast((resp as any)['errmsg'])
    }
    const find = items.value?.find(x => {
      return x.uuid == item.uuid
    })
    if (find && resp) {
      find.status.active = false
      find.status.enable = false
      toast('SUCCESS')
    }
  } catch (err) {
    console.error('get mcp:', err)
    return toast('ERROR:'+err)
  }
}

const doLoad = async () => {
  try {
    const url = `/mcp?act=get-mcp`
    const resp = await request.post(url)
    items.value = (resp as McpServer[]).filter((item) => {
      return item.uuid !== 'builtin'
    })
    items.value = items.value.map((item) => {
      if (!item.status) {
        item.status = {} as McpStatus
      }
      if (!item.status?.enable) {
        item.status.enable = false
      }
      return item
    })
    items.value.forEach((item: McpServer) => {
      if (!item.status) {
        item.status = {} as McpStatus
        item.status.enable = false
      }
      if (item.status.enable) {
        loading.value[item.uuid] = true
        return doActive(item.uuid)
      }
    });
  } catch (err) {
    console.error('get mcp:', err)
    return toast('ERROR:'+err)
  }
}

const doActive = async (uuid: string) => {
  try {
    loading.value[uuid] = true
    const url = `/mcp?act=active&uuid=${uuid}`
    const resp = await request.post(url) as McpStatus
    if ((resp as any)?.errmsg) {
      throw (resp as any)?.errmsg
    }
    const find = items.value?.find(x => {
      return x.uuid == uuid
    })
    if (find && resp.tools) {
      resp.enable = !!resp.enable
      find.status = resp as McpStatus
    }
    app.getMcpList.forEach(item => {
      if (item.uuid == uuid) {
        resp.enable = !!resp.enable
        item.status = resp as McpStatus
      }
    })
  } catch (err) {
    const find = items.value?.find(x => {
      return x.uuid == uuid
    })
    if (find && find.status) {
      find.status.active = false
    }
    console.error('get mcp:', err)
    return toast('ERROR:' + err)
  } finally {
    loading.value[uuid] = false
  }
}
const onMenuClick = () => {
  active.value = ''
}
</script>

<template>
  <SetHeader :title="$t('menu.mcpSet')"/>
  <div id="mcp-setting" class="set-view">
    <div id="mcp-menu" class="set-menu">
      <div style="display: flex; gap: 8px; align-items: center;">
        <button class="btn-add-new" @click="onCreate">
          {{ $t('common.addMcp') }}
        </button>
        <icon icon="square-box" size="large" 
          @click="onMenuClick"
        />
      </div>
      <BasicMenu
        :items="items"
        :keyby="'name'"
        @click="onSelect">
        <template v-slot="{ item }">
          <div class="item-header">
            <h5>{{ item.name }}</h5>
            <p v-if="!item.status.active">
              {{ $t('common.notAvailable') }}
            </p>
            <p v-else>
              {{ item.status.tools?.length || 0 }}
              {{ $t('common.toolsAvailable')}}
            </p>
          </div>
          <div class="item-action">
            <a class="btn-detail" 
              @click="onDetail(item)">
              {{ $t('common.view') }}
            </a>
            <icon  v-if="loading[item.name]"
              icon="icon-loading" size="mini"
            />
            <SwitchInput v-else
              :id="'switch-' + item.name"
              :model-value="item.status.enable"
              @change="onSwitch(item, item.status)"
            />
          </div>
        </template>
      </BasicMenu>
    </div>
    <div id="mcp-panel" class="set-main">
      <McpMarket v-if="!active"/>
      <McpToolTest v-else :config="current"/>
    </div>
  </div>
</template>

<style scoped>
@import url('@/styles/setting.css');

.mcp-item {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.mcp-type {
  font-size: 12px;
  color: #666;
  font-weight: normal;
  margin-bottom: 2px;
}

.mcp-content {
  font-size: 14px;
  color: inherit;
  font-weight: bold;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}


.btn-add-new{
  margin: 5px auto;
  font-weight: normal;
  width: -webkit-fill-available;
}
.btn-market {
  padding: 5px;
  min-width: 36px;
  min-height: 36px;
  transition: background 0.2s;
  border: 2px solid buttonface;
}
.btn-market:hover {
  background: var(--bg-menu);
}
</style>
