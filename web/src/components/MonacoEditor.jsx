import { Editor } from '@monaco-editor/react'

// 根据文件名判断 Monaco 语言
export function langFromName(name = '') {
  const ext = name.slice(name.lastIndexOf('.') + 1).toLowerCase()
  const map = {
    js: 'javascript', jsx: 'javascript', mjs: 'javascript', cjs: 'javascript',
    ts: 'typescript', tsx: 'typescript',
    json: 'json',
    html: 'html', htm: 'html',
    css: 'css', scss: 'scss', less: 'less',
    md: 'markdown',
    xml: 'xml', yml: 'yaml', yaml: 'yaml',
    sql: 'sql', sh: 'shell', bat: 'shell', ps1: 'powershell',
    py: 'python', go: 'go', java: 'java', c: 'c', cpp: 'cpp',
    rb: 'ruby', php: 'php', rs: 'rust', toml: 'ini', ini: 'ini', conf: 'ini',
  }
  return map[ext] || 'plaintext'
}

// Monaco 编辑器封装（@monaco-editor/react 加载 Monaco 并自动管理 Worker）
export default function MonacoEditor({ value, onChange, language, readOnly = false }) {
  return (
    <Editor
      height="100%"
      language={language || 'plaintext'}
      value={value || ''}
      onChange={onChange}
      theme="vs-dark"
      options={{
        readOnly,
        automaticLayout: true,
        minimap: { enabled: false },
        fontSize: 13,
        fontFamily: "'Cascadia Mono','Consolas',monospace",
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        tabSize: 2,
      }}
    />
  )
}
