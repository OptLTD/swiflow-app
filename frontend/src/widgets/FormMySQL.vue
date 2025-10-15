<script setup lang="ts">
import { PropType, ref, watch } from 'vue'

const props = defineProps({
  mysql: {
    type: Object as PropType<CfgMySQLMeta>,
    default: () => {}
  },
  errmsg: {
    type: String as PropType<String>,
    default: () => ''
  },
})

const theForm = ref<HTMLFormElement>()
const formModel = ref(props.mysql || {})
watch(() => props.mysql, (data) => {
  Object.assign(formModel.value, {...data})
})

const getFormModel = () => {
  const form = theForm.value
  if (form!.checkValidity()) {
    return formModel.value
  }
  return null
}

defineExpose({ getFormModel })
</script>

<template>
  <form ref="theForm">
    <h3 class="title">MySQL Connection</h3>
    <div class="form-item">
      <label>Host：</label>
      <input v-model="formModel.host" type="text" required/>
    </div>
    <div class="form-item">
      <label>Username：</label>
      <input v-model="formModel.user" type="text" required/>
    </div>
    <div class="form-item">
      <label>Password：</label>
      <input v-model="formModel.pass" type="password" required autocomplete="false"/>
    </div>
    <div class="form-item">
      <label>Database：</label>
      <input v-model="formModel.name" type="text" required/>
    </div>
    <div class="form-item">
      <label>Port：</label>
      <input v-model="formModel.port" placeholder="3306"/>
    </div>
    <div class="form-item">
      <label>Deny List：</label>
      <input v-model="formModel.deny" placeholder="users,deny_tables"/>
    </div>
    <div class="form-item err-msg" v-if="props.errmsg">
      {{ props.errmsg }}
    </div>
  </form>
</template>

<style scoped>
@import url('@/styles/form.css');
.title{
  text-align: center;
  margin-top: -15px;
  margin-bottom: 25px;
}

.form-item {
  display: flex;
  height: 32px;
  margin: 12px 12px;
  justify-content: space-between;
}
.form-item > label{
  width: 8rem;
  display: block;
  font-size: 1.1rem;
  font-weight: bold;
  text-align: right;
}
.form-item > input{
  flex: 1;
}
.form-item.err-msg{
  width: 100%;
  display: block;
  text-align: center;
}
</style>
