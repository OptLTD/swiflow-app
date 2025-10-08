import { isEmpty } from 'lodash-es';
import { toast } from 'vue3-toastify';
import { md, textType } from '@/support';
import { request, isHTML } from '@/support';
import csvToMarkdown from "csv-to-markdown-table";


// 伪代码示例
const isScrollToEnd = (container: HTMLElement) => {
  const threshold = 150
  const containerHeight = container.scrollTop + container.clientHeight
  return containerHeight >= container.scrollHeight - threshold
}

export const AUTO_DISPLAY = ['complete']

export const shouldAutoDisplay = (msg: MsgAct) => {
  return AUTO_DISPLAY.includes(msg.type)
}

export const canShowDisplayAct = (msg: MsgAct) => {
  if (!shouldAutoDisplay(msg)) {
    return false;
  }
  const data = (msg as Complete).content
  return data && data.split('\n').length > 5
}

export const autoScrollToEnd = (force: boolean) => {
  const container = document.querySelector('.list-container')
  if (!container) {
    return
  }
  if (!isScrollToEnd(container as HTMLElement) && !force) {
    return
  }
  container!.scrollTo({ top: container.scrollHeight, behavior: 'smooth' });
}

export const doReplayExecute = async (uuid: string, msgid: string) => {
  const url = `/execute?act=replay&uuid=${uuid}&msgid=${msgid}`
  const resp = await request.post(url) as any
  console.log('replay', uuid, resp)
  if (resp['errmsg'] != '') {
    toast.error(resp['errmsg'])
  } else {
    toast.success('success')
  }
}

export const setBot4Task = async (uuid: string, bot: BotEntity) => {
  try {
    const url = `/task?uuid=${uuid}&bot=${bot.uuid}`
    const resp = await request.post(url + '&act=set-bot')
    console.log('do use bot', resp, uuid)
  } catch (err) {
    console.error('use bot:', err)
  }
}

export const doUploadFiles = async (uuid: string, files: File[]): Promise<any> => {
  const url = `/upload?uuid=${uuid}`
  const data = new FormData()
  for (const file of files) {
    data.append('files', file, file.name)
  }
  const options = {headers: {},method: 'POST',body: data}
  return request.fetch(url, options).then((resp) => {
    return resp.json()
  }).catch(err => {
    console.log('error', err)
    return {errmsg: err}
  })
}

// Import .agent files to create new bots
export const doImportFiles = async (files: File[]): Promise<any> => {
  const url = `/import`
  const data = new FormData()
  for (const file of files) {
    data.append('files', file, file.name)
  }
  const options = {headers: {},method: 'POST',body: data}
  return request.fetch(url, options).then((resp) => {
    return resp.json()
  }).catch(err => {
    console.log('error', err)
    return {errmsg: err}
  })
}

export const setHomePath = async (uuid: string, path: string) => {
  if (uuid == "") {
    return
  }
  try {
    const args = `home=${path}&uuid=${uuid}`
    const url = `/bot?act=set-home&${args}`
    const resp = await request.post(url) as any
    if (resp && resp.errmsg) {
      return toast.error(resp.errmsg)
    }
  } catch (err) {
    console.error('set-home:', err)
  }
}

export const setBotTools = async (uuid: string, tools: string) => {
  if (uuid == "") {
    return
  }
  try {
    const url = `/bot?act=set-tools&uuid=${uuid}&tools=${tools}`
    const resp = await request.post(url, {tools}) as any
    if (!!resp.errmsg) {
      return toast.error(resp.errmsg)
    }
  } catch (err) {
    console.error('launch:', err)
  }
}

export const setBotProvider = async (uuid: string, name: string) => {
  if (uuid == "") {
    return
  }
  const url = `/bot?act=set-provider&uuid=${uuid}&provider=${name}`
  return request.post(url, { provider: name }) as any
}

export const parseHistory = (msgs: ActionMsg[]): ActionMsg[] => {
  const dict: Record<string, number[]> = {};

  // 第一步：收集每个hash的所有出现位置
  msgs.forEach((msg, idx) => {
    msg.actions.forEach((action, i) => {
      action.msgid = msg.theMsgId
      if (!dict[action.hash]) {
        dict[action.hash] = [];
      }
      dict[action.hash].push(idx * 1000 + i);
    });
  });

  for (const hash in dict) {
    const positions = dict[hash];

    if (positions.length <= 1) continue;

    for (let i = 0; i < positions.length - 1; i++) {
      const posi = positions[i];
      const msgIdx = Math.floor(posi / 1000);
      const actIdx = posi % 1000;

      msgs[msgIdx].actions[actIdx].hide = true;
    }

    const lastPosi = positions[positions.length - 1];
    const lastMsgIdx = Math.floor(lastPosi / 1000);
    const lastActIdx = lastPosi % 1000;
    msgs[lastMsgIdx].actions[lastActIdx].hide = false;
  }

  return msgs;
}

export const getDisplayActDesc = (item: MsgAct): string => {
  switch (item.type) {
    case "complete": {
      return `结果展示`
    }
    case "execute-command": {
      const act = (item as ExecuteCommand)
      return `执行命令: ${act.command || ''}`
    }
    case "start-async-cmd": {
      const act = (item as StartAsyncCmd)
      return `执行异步命令: ${act.session}(${act.command})`
    }
    case "query-async-cmd": {
      const act = (item as QueryAsyncCmd)
      return `查询异步命令: ${act.session}`
    }
    case "abort-async-cmd": {
      const act = (item as AbortAsyncCmd)
      return `终止异步命令: ${act.session}`
    }
    case "start-subtask": {
      const act = (item as StartSubtask)
      return `执行子任务: ${act['task-desc']}`
    }
    case "query-subtask": {
      const act = (item as QuerySubtask)
      return `查询子任务: ${act['sub-agent']}`
    }
    case "abort-subtask": {
      const act = (item as AbortSubtask)
      return `终止子任务: ${act['sub-agent']}`
    }
    case 'path-list-files': {
      const act = (item as PathListFiles)
      return `列取目录: ${act.path || ''}`
    }
    case 'file-get-content': {
      const act = (item as FileGetContent)
      return `查看文件: ${act.path || ''}`
    }
    case 'file-replace-text':
    case 'file-put-content': {
      const act = (item as FilePutContent)
      return `修改文件: ${act.path || ''}`
    }
    case "use-mcp-tool": {
      const act = (item as UseMcpTool)
      return `Use Mcp: ${act.desc}`
    }
    case "use-builtin-tool": {
      const act = (item as UseBuiltinTool)
      return `内置工具(${act.tool}): ${act.desc}`
    }
  }
  return `undefined ${item.type}`
}

export const getActHtml = (data: MsgAct) => {
  if (!data) {
    return ''
  }
  const { errmsg, result } = (data as DefaultResult)
  if (!isEmpty(errmsg)) {
    const content = '```plain\n{{ERRMSG}}\n```\n'
    return md.render(content.replace('{{ERRMSG}}', errmsg))
  }
  // mcp tool result
  const tools = ['use-mcp-tool', 'use-builtin-tool']
  if (tools.includes(data.type) && !isEmpty(result)) {
    const parts = [] as string[]
    const type = textType(result)
    const { args, more } = (data as UseMcpTool)
    if (args.trim() && more === true) {
      const trim = args.replace(/(\n\s{4})/ig, '\n').trim()
      parts.push('```json', trim, '```', '---')
    }
    // console.log('log', type, result)
    switch (type) {
      case 'html':
        return md.render(parts.join('\n')) + result
      case 'markdown':
        parts.push(result); break;
      case 'csv':
        parts.push(csvToMarkdown(result, ',', true)); break;
      case 'json':
        const data = JSON.stringify(JSON.parse(result), null, 2) 
        parts.push('```json\n' + data + '\n```'); break;
      default:
        parts.push('```plain\n' + result + '\n```'); break;
    }
    return md.render(parts.join('\n'))
  }

  const cmds = [
    'execute-command',
    'start-async-cmd',
    'query-async-cmd',
    'abort-async-cmd',
  ]
  if (cmds.includes(data.type)) {
    return md.render('```sh\n' + result.trim() + '\n```')
  }

  // self tool result
  const dev = ['.js', '.py', '.php', '.java']
  switch (data.type) {
    case "complete": {
      const act = (data as DefaultAction)
      if (isHTML(act.content)) {
        return act.content
      }
      return md.render(act.content)
    }
    case 'path-list-files': {
      if (!isEmpty(result)) {
        return md.render(result)
      }
      return md.render('empty result')
    }
    case 'file-replace-text': {
      const act = (data as FileReplaceText)
      var content = '### 修改内容\n\n```plain\n{{DIFF}}\n```\n\n'
      content += '### 修改结果\n\n```{{SUFFIX}}\n{{RESULT}}\n```'
      if (dev.some((x => act.path.endsWith(x)))) {
        const suffix = act.path.split('.').pop() as string
        content = content.replace('{{SUFFIX}}', suffix)
      } else {
        content = content.replace('{{SUFFIX}}', 'plain')
      }
      content = content.replace('{{DIFF}}', act.diff || '')
      content = content.replace('{{RESULT}}', result || '')
      return md.render(content)
    }
    case 'file-get-content':
    case 'file-put-content': {
      const act = (data as FilePutContent)
      var content = '```{{SUFFIX}}\n{{RESULT}}\n```\n'
      content = content.replace('{{RESULT}}', result.trim())
      if (dev.some((x => act.path.endsWith(x)))) {
        const suffix = act.path.split('.').pop() as string
        content = content.replace('{{SUFFIX}}', suffix)
      } else if (!act.path.endsWith('.md')) {
        content = content.replace('{{SUFFIX}}', 'plain')
      } else {
        content = result
      }
      return md.render(content.trim())
    }
    case 'start-subtask': {
      const act = (data as StartSubtask)
      var content = '### Task Desc\n\n```plain\n{{TASKDESC}}\n```\n\n'
      content += '### Context\n\n{{CONTEXT}}\n\n'
      content += '### Require\n\n{{REQUIRE}}\n\n'
      content = content.replace('{{TASKDESC}}', act['task-desc'] || '')
      content = content.replace('{{CONTEXT}}', act.context || '')
      content = content.replace('{{REQUIRE}}', act.require || '')
      return md.render(content)
    }
  }
  return ''
}