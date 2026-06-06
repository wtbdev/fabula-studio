import {
  AlignmentType,
  BorderStyle,
  convertInchesToTwip,
  Document,
  Footer,
  Header,
  PageNumber,
  Packer,
  Paragraph,
  ShadingType,
  TextRun,
} from 'docx'
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

const FONT_SANS = 'Noto Sans SC'
const FONT_SERIF = 'Noto Serif SC'
const COLOR_INK = '17211f'
const COLOR_MUTED = '596966'
const COLOR_SAGE = '2f7664'
const COLOR_LINE = 'dce3df'
const TWIP = convertInchesToTwip

const PAGE_MARGIN = {
  top: TWIP(1),
  bottom: TWIP(1),
  left: TWIP(1.4),
  right: TWIP(1),
}

function sceneHeading(scene: SceneDTO): Paragraph {
  const parts: string[] = []
  if (scene.location) parts.push(scene.location)
  if (scene.timeText) parts.push(scene.timeText)
  const slugline = parts.join(' - ')

  return new Paragraph({
    spacing: { before: 360, after: 160 },
    border: {
      bottom: { style: BorderStyle.SINGLE, size: 4, color: COLOR_LINE, space: 6 },
    },
    children: [
      new TextRun({
        text: `第 ${scene.sceneNo} 场`,
        font: FONT_SANS,
        size: 20,
        color: COLOR_SAGE,
        bold: true,
      }),
      new TextRun({ text: '    ', font: FONT_SANS, size: 20 }),
      ...(slugline
        ? [
            new TextRun({
              text: slugline.toUpperCase(),
              font: FONT_SANS,
              size: 22,
              bold: true,
              color: COLOR_INK,
            }),
          ]
        : []),
      new TextRun({ text: '    ', font: FONT_SANS, size: 20 }),
      new TextRun({
        text: scene.title,
        font: FONT_SERIF,
        size: 22,
        color: COLOR_MUTED,
      }),
    ],
  })
}

function sceneSummaryParagraph(summary: string): Paragraph {
  return new Paragraph({
    spacing: { before: 80, after: 160 },
    indent: { left: TWIP(0.3) },
    border: {
      left: { style: BorderStyle.SINGLE, size: 8, color: COLOR_SAGE, space: 8 },
    },
    shading: { type: ShadingType.CLEAR, fill: 'eef7f3' },
    children: [
      new TextRun({ text: summary, font: FONT_SERIF, size: 21, color: COLOR_MUTED, italics: true }),
    ],
  })
}

function actionParagraph(text: string): Paragraph {
  return new Paragraph({
    spacing: { after: 140 },
    children: [new TextRun({ text, font: FONT_SERIF, size: 24, color: COLOR_INK })],
  })
}

function dialogueBlock(character: string, text: string): Paragraph[] {
  return [
    new Paragraph({
      spacing: { before: 200, after: 40 },
      alignment: AlignmentType.CENTER,
      children: [
        new TextRun({
          text: character.toUpperCase(),
          font: FONT_SANS,
          size: 22,
          bold: true,
          color: COLOR_INK,
        }),
      ],
    }),
    new Paragraph({
      spacing: { after: 140 },
      indent: { left: TWIP(1.2), right: TWIP(1.2) },
      alignment: AlignmentType.CENTER,
      children: [new TextRun({ text, font: FONT_SERIF, size: 24, color: COLOR_INK })],
    }),
  ]
}

function transitionParagraph(text: string): Paragraph {
  return new Paragraph({
    spacing: { before: 240, after: 200 },
    alignment: AlignmentType.RIGHT,
    children: [
      new TextRun({
        text: text.toUpperCase(),
        font: FONT_SANS,
        size: 22,
        bold: true,
        color: COLOR_MUTED,
      }),
    ],
  })
}

function sceneSeparator(): Paragraph {
  return new Paragraph({
    spacing: { before: 200, after: 200 },
    alignment: AlignmentType.CENTER,
    border: {
      bottom: { style: BorderStyle.SINGLE, size: 2, color: COLOR_LINE, space: 8 },
    },
    children: [],
  })
}

function buildDocx(project: ProjectDTO, scenes: SceneDTO[], activeSceneId: string | null, editorContent: string): Document {
  const sorted = sortedScenes(scenes, activeSceneId, editorContent)
  const now = new Date()
  const dateStr = `${now.getFullYear()} 年 ${now.getMonth() + 1} 月 ${now.getDate()} 日`

  const titleChildren: Paragraph[] = [
    new Paragraph({ spacing: { before: 3600 }, children: [] }),
    new Paragraph({
      alignment: AlignmentType.CENTER,
      spacing: { after: 200 },
      children: [new TextRun({ text: project.title, font: FONT_SERIF, size: 56, bold: true, color: COLOR_INK })],
    }),
  ]

  if (project.novelTitle) {
    titleChildren.push(
      new Paragraph({
        alignment: AlignmentType.CENTER,
        spacing: { after: 120 },
        children: [
          new TextRun({ text: '原著：', font: FONT_SANS, size: 24, color: COLOR_MUTED }),
          new TextRun({ text: project.novelTitle, font: FONT_SERIF, size: 24, color: COLOR_MUTED }),
        ],
      }),
    )
  }

  titleChildren.push(
    new Paragraph({
      alignment: AlignmentType.CENTER,
      spacing: { before: 600 },
      children: [new TextRun({ text: '叙幕工作室 生成', font: FONT_SANS, size: 20, color: COLOR_SAGE })],
    }),
    new Paragraph({
      alignment: AlignmentType.CENTER,
      spacing: { after: 200 },
      children: [new TextRun({ text: dateStr, font: FONT_SANS, size: 20, color: COLOR_MUTED })],
    }),
    new Paragraph({ spacing: { before: 2400 }, children: [] }),
  )

  const mainChildren: Paragraph[] = []

  for (const scene of sorted) {
    mainChildren.push(sceneHeading(scene))

    if (scene.summary) {
      mainChildren.push(sceneSummaryParagraph(scene.summary))
    }

    if (scene.rawJson?.script && scene.rawJson.script.length > 0) {
      for (const block of scene.rawJson.script) {
        switch (block.type) {
          case 'dialogue':
            mainChildren.push(...dialogueBlock(block.character ?? '', block.content))
            break
          case 'narration':
          case 'voice_over':
            mainChildren.push(
              new Paragraph({
                spacing: { after: 140 },
                indent: { left: TWIP(0.4) },
                children: [
                  new TextRun({
                    text: block.type === 'voice_over' ? '画外音：' : '旁白：',
                    font: FONT_SANS,
                    size: 21,
                    bold: true,
                    color: COLOR_SAGE,
                  }),
                  new TextRun({ text: block.content, font: FONT_SERIF, size: 23, color: COLOR_MUTED, italics: true }),
                ],
              }),
            )
            break
          case 'transition':
            mainChildren.push(transitionParagraph(block.content))
            break
          default:
            mainChildren.push(actionParagraph(block.content))
            break
        }
      }
    } else {
      const lines = scene.content.split('\n').filter((l) => l.trim())
      for (const line of lines) {
        mainChildren.push(actionParagraph(line))
      }
    }

    mainChildren.push(sceneSeparator())
  }

  if (mainChildren.length > 0) {
    mainChildren.pop()
  }

  const pageHeader = new Header({
    children: [
      new Paragraph({
        alignment: AlignmentType.RIGHT,
        children: [
          new TextRun({ text: project.title, font: FONT_SANS, size: 16, color: COLOR_LINE }),
        ],
      }),
    ],
  })

  const pageFooter = new Footer({
    children: [
      new Paragraph({
        alignment: AlignmentType.CENTER,
        children: [
          new TextRun({ text: '- ', font: FONT_SANS, size: 18, color: COLOR_MUTED }),
          new TextRun({ children: [PageNumber.CURRENT], font: FONT_SANS, size: 18, color: COLOR_MUTED }),
          new TextRun({ text: ' -', font: FONT_SANS, size: 18, color: COLOR_MUTED }),
        ],
      }),
    ],
  })

  return new Document({
    styles: {
      default: {
        document: {
          run: { font: FONT_SANS, size: 24, color: COLOR_INK },
          paragraph: { spacing: { line: 360 } },
        },
      },
    },
    sections: [
      {
        properties: {
          page: { margin: PAGE_MARGIN },
          titlePage: true,
        },
        children: titleChildren,
      },
      {
        properties: {
          page: { margin: PAGE_MARGIN },
        },
        headers: { default: pageHeader },
        footers: { default: pageFooter },
        children: mainChildren,
      },
    ],
  })
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
    const doc = buildDocx(project, scenes, activeSceneId, editorContent)
    const blob = await Packer.toBlob(doc)
    triggerDownload(blob, filename)
    return
  }

  const content = buildExportContent(format, project, scenes, activeSceneId, editorContent)
  triggerDownload(
    new Blob([content], { type: 'text/plain;charset=utf-8' }),
    filename,
  )
}
