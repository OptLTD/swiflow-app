import markdownit from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import markdown from 'highlight.js/lib/languages/markdown'
import python from 'highlight.js/lib/languages/python'
import bash from 'highlight.js/lib/languages/bash'
import json from 'highlight.js/lib/languages/json'
import yaml from 'highlight.js/lib/languages/yaml'
import sql from 'highlight.js/lib/languages/sql'
import php from 'highlight.js/lib/languages/php'
import java from 'highlight.js/lib/languages/java'
import css from 'highlight.js/lib/languages/css'
import xml from 'highlight.js/lib/languages/xml'

import { toast } from 'vue3-toastify'
import 'highlight.js/styles/vs.min.css';
import { tasklist } from "@mdit/plugin-tasklist";

import { BASE_ADDR } from './consts' 
import { XMLParser } from './parser'
import { HttpRequest } from './request'

export * as errors from './errors'
export { toast }

export const parser = new XMLParser()

export const request = new HttpRequest(BASE_ADDR);

export const isTauri = () => {
  return !!((globalThis as any) || window).isTauri
}
export const isHTML = (str: string) =>  {
  try {
    const parser = new DOMParser();
    const doc = parser.parseFromString(str, 'text/html');
    return Array.from(doc.body.childNodes).some(node => {
      switch (node.nodeName) {
        case 'COMMAND':
          return false;
      }
      return node.nodeType === 1
    });
  } catch (e) {
    return false;
  }
}
export const isJSON = (str: string) => {
  if (typeof str !== 'string') return false;
  try {
    const obj = JSON.parse(str);
    // 只接受对象或数组作为有效JSON
    return typeof obj === 'object' && obj !== null;
  } catch (e) {
    return false;
  }
}

// 判断是否为Markdown格式
export const isMarkdown = (str: string) => {
  if (typeof str !== 'string') return false;
  const markdownPatterns = [
    /^#{1,6} /m,                // 标题
    /^([-*+]|\d+\.) /m,         // 列表
    /!\[.*?\]\(.*?\)/,          // 图片
    /\[.*?\]\(.*?\)/,           // 链接
    /`{1,3}[^`]+`{1,3}/,        // 行内/多行代码
    /^> /m,                     // 引用
    /^---$|^\*\*\*$|^___$/m,    // 分割线
  ];
  return markdownPatterns.some(pattern => pattern.test(str));
}

// 判断是否为CSV格式
export const isCSV = (str: string) => {
  if (typeof str !== 'string') return false;
  // 检查是否有多行，且每行逗号数量大致相同
  const lines = str.trim().split(/\r?\n/);
  if (lines.length < 2) return false;
  const commaCounts = lines.map(line => (line.match(/,/g) || []).length);
  const firstCount = commaCounts[0];
  // 允许部分行为空，但大部分行逗号数应一致
  const similarCount = commaCounts.filter(c => c === firstCount).length;
  return similarCount >= Math.floor(lines.length * 0.7) && firstCount > 0;
}

export const textType = (str: string) => {
  if (isJSON(str)) {
    return 'json'
  }
  if (isCSV(str)) {
    return 'csv'
  }
  if (isMarkdown(str)) {
    return 'markdown'
  }
  if (isHTML(str)) {
    return 'html'
  }
  return 'text'
}

export const md = markdownit({
  highlight: function (str: string, lang: string) {
    if (lang) {
      // lazy register common languages once
      if (!hljs.listLanguages().length) {
        hljs.registerLanguage('javascript', javascript)
        hljs.registerLanguage('typescript', typescript)
        hljs.registerLanguage('python', python)
        hljs.registerLanguage('bash', bash)
        hljs.registerLanguage('json', json)
        hljs.registerLanguage('markdown', markdown)
        hljs.registerLanguage('yaml', yaml)
        hljs.registerLanguage('sql', sql)
        hljs.registerLanguage('php', php)
        hljs.registerLanguage('java', java)
        hljs.registerLanguage('css', css)
        hljs.registerLanguage('xml', xml)
      }
    }
    if (lang && hljs.getLanguage(lang)) {
      try {
        const conf = { language: lang, ignoreIllegals: true }
        const html = hljs.highlight(str, conf).value
        return `<pre><code class="hljs">${html}</code></pre>`;
      } catch (__) {}
    }
    const escape = md.utils.escapeHtml(str) as string
    return `<pre><code class="hljs">${escape}</code></pre>`;
  }
}).use(linkTargetBlank).use(tasklist, {});

function linkTargetBlank(md: any) {
  // @ts-ignore
  const defaultRender = md.renderer.rules.link_open || function(tokens, idx, options, env, self) {
    return self.renderToken(tokens, idx, options);
  };

  // @ts-ignore
  md.renderer.rules.link_open = function(tokens, idx, options, env, self) {
    // target="_blank"
    const aIndex = tokens[idx].attrIndex('target');
    if (aIndex < 0) {
      tokens[idx].attrPush(['target', '_blank']);
    } else {
      tokens[idx].attrs[aIndex][1] = '_blank';
    }
    // rel="noopener noreferrer"
    const relIndex = tokens[idx].attrIndex('rel');
    if (relIndex < 0) {
      tokens[idx].attrPush(['rel', 'noopener noreferrer']);
    } else {
      tokens[idx].attrs[relIndex][1] = 'noopener noreferrer';
    }
    return defaultRender(tokens, idx, options, env, self);
  };
}

export const confirm = (tips: string) => {
  // 使用标准的window.confirm，只传递消息字符串
  return window.confirm(tips)
}

export const alert = (tips: string) => {
  return toast.info(tips)
}

export const error = (tips: string) => {
  return toast.error(tips)
}



