import type { ProjectDTO, SceneDTO } from '../api/types'

export type ExportFormat = 'txt' | 'markdown' | 'yaml' | 'word'

export interface ExportFormatOption {
  key: ExportFormat
  label: string
  description: string
  extension: string
}

export const exportFormats: ExportFormatOption[] = [
  {
    key: 'txt',
    label: '纯文本',
    description: '简洁的 .txt 文件，适合通用阅读和打印',
    extension: 'txt',
  },
  {
    key: 'word',
    label: 'Word',
    description: '带排版的 .docx 文件，适合正式交付和协作审阅',
    extension: 'docx',
  },
  {
    key: 'markdown',
    label: 'Markdown',
    description: '结构化 .md 文件，支持标题层级和格式化',
    extension: 'md',
  },
  {
    key: 'yaml',
    label: 'YAML',
    description: '结构化剧本 Schema，适合程序解析和版本管理',
    extension: 'yaml',
  },
]

const safeTitle = (title: string) => title.replace(/[\\/:*?"<>|]/g, '_')

const sortedScenes = (scenes: SceneDTO[], activeSceneId: string | null, editorContent: string) =>
  scenes
    .map((scene) =>
      scene.id === activeSceneId ? { ...scene, content: editorContent } : scene,
    )
    .sort((a, b) => a.sceneNo - b.sceneNo)

function buildTxt(scenes: SceneDTO[], activeSceneId: string | null, editorContent: string): string {
  const sorted = sortedScenes(scenes, activeSceneId, editorContent)
  return sorted.map((scene) => scene.content).join('\n\n---\n\n')
}

function buildMarkdown(project: ProjectDTO, scenes: SceneDTO[], activeSceneId: string | null, editorContent: string): string {
  const sorted = sortedScenes(scenes, activeSceneId, editorContent)
  const lines: string[] = []

  lines.push(`# ${project.title}`)
  lines.push('')

  if (project.novelTitle) {
    lines.push(`> 原著：${project.novelTitle}`)
    lines.push('')
  }

  lines.push('---')
  lines.push('')

  for (const scene of sorted) {
    lines.push(`## 第 ${scene.sceneNo} 场：${scene.title}`)
    lines.push('')

    const metaParts: string[] = []
    if (scene.location) metaParts.push(`📍 ${scene.location}`)
    if (scene.timeText) metaParts.push(`🕐 ${scene.timeText}`)
    if (metaParts.length > 0) {
      lines.push(`*${metaParts.join('  |  ')}*`)
      lines.push('')
    }

    if (scene.summary) {
      lines.push(`> ${scene.summary}`)
      lines.push('')
    }

    lines.push(scene.content)
    lines.push('')
    lines.push('---')
    lines.push('')
  }

  return lines.join('\n')
}

const yamlEscape = (value: string): string => {
  if (!value) return '""'
  if (/[:{}\[\],&*?|>!%@`#'"\n\r]/.test(value) || /^\s|\s$/.test(value)) {
    return `"${value.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\n/g, '\\n').replace(/\r/g, '\\r')}"`
  }
  return value
}

function buildYaml(project: ProjectDTO, scenes: SceneDTO[], activeSceneId: string | null, editorContent: string): string {
  const sorted = sortedScenes(scenes, activeSceneId, editorContent)
  const lines: string[] = []

  lines.push('screenplay:')
  lines.push('  metadata:')
  lines.push(`    title: ${yamlEscape(project.title)}`)
  if (project.novelTitle) {
    lines.push(`    original_novel: ${yamlEscape(project.novelTitle)}`)
  }
  lines.push(`    created_at: ${yamlEscape(new Date().toISOString())}`)
  lines.push('')

  const allCharacters = new Set<string>()
  for (const scene of sorted) {
    if (scene.rawJson?.characters) {
      scene.rawJson.characters.forEach((c) => allCharacters.add(c))
    }
  }

  if (allCharacters.size > 0) {
    lines.push('  characters:')
    for (const name of allCharacters) {
      lines.push(`    - name: ${yamlEscape(name)}`)
    }
    lines.push('')
  }

  lines.push('  scenes:')
  for (const scene of sorted) {
    lines.push(`    - id: ${yamlEscape(scene.id)}`)
    lines.push(`      sequence: ${scene.sceneNo}`)
    lines.push(`      heading: ${yamlEscape(scene.title)}`)
    lines.push('      setting:')
    lines.push(`        location: ${yamlEscape(scene.location ?? '')}`)
    lines.push(`        time: ${yamlEscape(scene.timeText ?? '')}`)
    if (scene.summary) {
      lines.push(`      synopsis: ${yamlEscape(scene.summary)}`)
    }

    if (scene.rawJson?.characters && scene.rawJson.characters.length > 0) {
      lines.push(`      characters_present: [${scene.rawJson.characters.map(yamlEscape).join(', ')}]`)
    }

    if (scene.rawJson?.script && scene.rawJson.script.length > 0) {
      lines.push('      content:')
      for (const block of scene.rawJson.script) {
        lines.push(`        - type: ${block.type}`)
        if (block.character) {
          lines.push(`          character: ${yamlEscape(block.character)}`)
        }
        lines.push(`          text: ${yamlEscape(block.content)}`)
      }
    } else {
      lines.push('      content:')
      lines.push(`        - type: action`)
      lines.push(`          text: ${yamlEscape(scene.content)}`)
    }

    lines.push('')
  }

  return lines.join('\n')
}


export function buildExportContent(
  format: Exclude<ExportFormat, 'word'>,
  project: ProjectDTO,
  scenes: SceneDTO[],
  activeSceneId: string | null,
  editorContent: string,
): string {
  switch (format) {
    case 'txt':
      return buildTxt(scenes, activeSceneId, editorContent)
    case 'markdown':
      return buildMarkdown(project, scenes, activeSceneId, editorContent)
    case 'yaml':
      return buildYaml(project, scenes, activeSceneId, editorContent)
  }
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

export async function downloadExport(
  format: ExportFormat,
  project: ProjectDTO,
  scenes: SceneDTO[],
  activeSceneId: string | null,
  editorContent: string,
) {
  const option = exportFormats.find((f) => f.key === format)!
  const filename = `${safeTitle(project.title) || '剧本工程'}.${option.extension}`

  if (format === 'word') {
    const { downloadWordExport } = await import('./wordExport')
    await downloadWordExport(project, scenes, activeSceneId, editorContent, filename)
    return
  }

  const content = buildExportContent(format, project, scenes, activeSceneId, editorContent)
  triggerDownload(
    new Blob([content], { type: 'text/plain;charset=utf-8' }),
    filename,
  )
}
