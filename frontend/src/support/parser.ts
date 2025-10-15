import { isArray } from "lodash-es";



export class XMLParser {
  public static readonly COMPLETE = 'complete';
  private static readonly MAKE_ASK = 'make-ask';
  private static readonly THINKING = 'thinking';
  private static readonly ANNOTATE = 'annotate';
  // private static readonly MEMORIZE = 'memorize';

  private static readonly START_SUBTASK = "start-subtask"
  private static readonly QUERY_SUBTASK = "query-subtask"
  private static readonly ABORT_SUBTASK = "abort-subtask"
  private static readonly EXECUTE_COMMAND = 'execute-command';
  private static readonly START_ASYNC_CMD = "start-async-cmd"
  private static readonly QUERY_ASYNC_CMD = "query-async-cmd"
  private static readonly ABORT_ASYNC_CMD = "abort-async-cmd"
  private static readonly PATH_LIST_FILES = 'path-list-files';
  private static readonly FILE_GET_CONTENT = 'file-get-content';
  private static readonly FILE_PUT_CONTENT = 'file-put-content';
  private static readonly FILE_REPLACE_TEXT = 'file-replace-text';


  private static readonly USE_MCP_TOOL = 'use-mcp-tool';
  private static readonly USE_BUILTIN_TOOL = 'use-builtin-tool';



  public Parse(data: string, worker?: string, msgid?: string): MsgResp {
    var msg: MsgResp = new MsgResp(data, worker, msgid);
    const tagReg = /<([^/>]+)>/;
    const maxAttempts = 10;
    let [left, curr, attempt] = [data, '', 0];
    while (left !== '' && attempt++ < maxAttempts) {
      const matches = left.match(tagReg);
      if (!matches?.length) {
        if (!left.trim().startsWith('<')) {
          var reply = trimBotReply(left);
          var action = new BotReply(reply);
          msg.actions.push(action as MsgAct);
        }
        break;
      }

      // get bot reply, tool will be
      if (left.indexOf(matches[0]) > -1) {
        var start = left.indexOf(matches[0])
        var before = left.substring(0, start)
        if (before.trim() != '') {
          left = left.substring(start)
          var reply = trimBotReply(before);
          var action = new BotReply(reply);
          msg.actions.push(action as MsgAct);
        }
      }

      [left, curr] = this.snap(left, matches[1]);
      if (!curr) continue;
      const res = this.parse(curr);
      if (!res) continue;
      if (res instanceof Annotate) {
        msg.context = res;
        continue;
      }
      if (res instanceof Thinking) {
        msg.thinking = res.content;
        continue;
      }
      if (res instanceof Error) {
        msg.errors.push(res);
        continue;
      }
      msg.actions.push(res);
    }
    return msg;
  }

  private snap(text: string, tag: string): [string, string] {
    const regex = new RegExp(`<${tag}>(.*?)(</${tag}>|$)`, 'sg');
    const result = regex.exec(text);
    if (!result) return [text, ''];

    const newText = text.replace(result[0], '');
    return [newText.trim(), result[0].trim()];
  }

  private parse(text: string): Error | MsgAct | null {
    if (!text.trim()) return null;

    const tagreg = /<([^/>]+)>/;
    const matches = text.match(tagreg);
    if (!matches || !matches[1]) {
      return new Error('unexpected data');
    }
    const theTag = matches[1] as string
    const inner = this.parseInner(theTag, text) as string
    const result = this.parseAction(inner.trim()) as any
    if (result == null) {
      return null
    }
    result['type'] = theTag
    switch (theTag) {
      case XMLParser.THINKING:
        if (!result['content']) { return null }
        return new Thinking(result['content']) as MsgAct
      case XMLParser.ANNOTATE: {
        if (!result['subject'] && !result['context']) {
          return null
        }
        const detail = new Annotate()
        detail.subject = result['summary']
        detail.context = result['todo-list']
        return detail as MsgAct
      }
      case XMLParser.COMPLETE: {
        if (!result['content']) {
          return null
        }
        return result as MsgAct
      }
      case XMLParser.MAKE_ASK: {
        if (!result['question']) { return null }
        const detail = new MakeAsk(result['question'])
        const options = this.parseAction(result['options'])
        detail.options = (options as any).option as string[]
        return detail as MsgAct
      }
      case XMLParser.PATH_LIST_FILES:
      case XMLParser.FILE_GET_CONTENT:
      case XMLParser.FILE_PUT_CONTENT:
      case XMLParser.FILE_REPLACE_TEXT: {
        if (!result['path']) {
          return null
        }
        return result as MsgAct
      }
      case XMLParser.EXECUTE_COMMAND: {
        if (!result['command']) {
          return null
        }
        return result as MsgAct
      }
      case XMLParser.START_SUBTASK:
      case XMLParser.QUERY_SUBTASK:
      case XMLParser.ABORT_SUBTASK: {
        if (!result['session']) {
          return null
        }
        return result as MsgAct
      }
      case XMLParser.USE_MCP_TOOL: {
        if (!result['tool']) { return null }
        if (!result['server']) { return null }
        const detail = new UseMcpTool(
          result['tool'], result['server']
        )
        detail.desc = result['desc']
        return detail as MsgAct
      }
      case XMLParser.USE_BUILTIN_TOOL: {
        if (!result['tool']) { return null }
        const detail = new UseBuiltinTool(
          result['tool'],
        )
        detail.desc = result['title']
        return detail as MsgAct
      }
      case XMLParser.START_ASYNC_CMD:
      case XMLParser.QUERY_ASYNC_CMD:
      case XMLParser.ABORT_ASYNC_CMD: {
        if (!result['session']) {
          return null
        }
        return result as MsgAct
      }
      default: {
        if (!result['content']) {
          return null
        }
        return result as MsgAct
      }
    }
  }

  private parseInner(tag: string, text: string): string {
    const tagreg = `<${tag}>([\\s\\S]*?)(?:<\\/${tag}>|$)`
    const matches = text.match(new RegExp(tagreg, 'i'));
    return matches ? matches[1] : ''
  }

  private parseAction(text: string): Error | MsgAct | null {
    const result = {} as any
    const maxAttempts = 10;
    const tagreg = /<([^/>]+)>/i;
    let [left, curr, attempt] = [text, '', 0];
    while (left && attempt++ < maxAttempts) {
      const matches = left.match(tagreg);
      if (!matches && attempt == 1) {
        result['content'] = left
        break
      }

      var [_, tagname] = matches || [, '']
      if (!matches || !tagname) continue;
      [left, curr] = this.snap(left, tagname);
      if (!curr) continue;
      // assin to result
      var content = this.parseInner(tagname, curr)
      if (result[tagname] && !isArray(result[tagname])) {
        result[tagname] = [result[tagname], content]
      } else if (result[tagname] && isArray(result[tagname])) {
        result[tagname].push(content)
      } else {
        result[tagname] = content
      }
    }
    return result;
  }
}

export class MsgResp {
  origin: string;
  context?: Annotate;
  errors: Error[] = [];
  actions: MsgAct[] = [];
  thinking?: string;
  datetime?: string;
  theMsgId?: string;
  workerId?: string;

  constructor(data: string, worker?: string, msgid?: string) {
    this.origin = data;
    this.thinking = '';
    this.theMsgId = msgid
    this.workerId = worker
  }
}


class MakeAsk {
  question: string;
  options: string[] = [];
  constructor(question: string) {
    this.question = question
  }
}

class Annotate {
  subject: string = '';
  context: string = '';
  constructor() {
  }
}

class Thinking {
  content: string;
  constructor(content: string) {
    this.content = content
  }
}

class BotReply {
  content: string;
  constructor(content: string) {
    this.content = content
  }
}


class UseMcpTool {
  desc?: string;
  tool: string;
  name: string;
  args?: Record<string, any>;
  constructor(a: string, b: string) {
    this.tool = a
    this.name = b
  }
}

class UseBuiltinTool {
  tool: string;
  desc?: string;
  args?: any;
  constructor(name: string) {
    this.tool = name
  }
}

function trimBotReply(text: string): string {
  text = text.trim();
  if (text.endsWith('```xml')) {
    text = text.slice(0, -'```xml'.length);
    text = text.trim();
  }
  if (text.startsWith('```')) {
    text = text.slice('```'.length);
    text = text.trim();
  }
  if (text == '<--[tool-result]-->') {
    return ''
  }
  return text;
}

