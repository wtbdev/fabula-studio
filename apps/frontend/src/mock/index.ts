import Mock from 'mockjs'
import { authTokenStorageKey } from '../api/request'
import type {
  AdaptConfig,
  ApiResponse,
  AuthTokenDTO,
  CreateProjectRequest,
  GenerateSceneRegenerationRequest,
  GenerateSceneRegenerationResponse,
  GenerateSceneSuggestionsRequest,
  GenerateSceneSuggestionsResponse,
  GenerateProjectResponse,
  GenerateStatusDTO,
  LoginRequest,
  ProjectDTO,
  RegisterRequest,
  SceneDTO,
  SceneRegenerationMode,
  SceneSuggestion,
  SceneSuggestionStatus,
  UpdateProjectRequest,
  UpdateSceneRequest,
  UpdateSceneResponse,
  UpdateSceneSuggestionRequest,
  UpdateSceneSuggestionResponse,
  UserDTO,
} from '../api'

interface MockRequestOptions {
  url: string
  type: string
  body: string
}

interface MockUser extends UserDTO {
  password: string
}

const now = () => new Date().toISOString()

const ok = <T>(data: T, message = 'success'): ApiResponse<T> => ({
  code: 0,
  message,
  data,
})

const fail = (code: number, message: string): ApiResponse<null> => ({
  code,
  message,
  data: null,
})

const parseBody = <T>(options: MockRequestOptions): T => {
  if (!options.body) return {} as T

  try {
    return JSON.parse(options.body) as T
  } catch {
    return {} as T
  }
}

const toUrl = (url: string) => new URL(url, window.location.origin)

const getPathSegment = (options: MockRequestOptions, index: number) => {
  return toUrl(options.url).pathname.split('/').filter(Boolean)[index] ?? ''
}

const createId = (prefix: string) => `${prefix}_${Mock.Random.string('lower', 8)}`

const defaultConfig: AdaptConfig = {
  style: '影视剧',
  dialogueLevel: '适中',
  adaptationMode: '忠实原文',
  sceneGranularity: '适中',
  narrationLevel: '少量保留',
  customPrompt: '减少旁白，增强对白冲突。',
}

const users: MockUser[] = [
  {
    id: 'user_001',
    email: 'writer@example.com',
    password: '123456',
    nickname: '创作者用户',
    aiPoints: 1000,
    createdAt: '2026-06-06T10:00:00.000Z',
    updatedAt: '2026-06-06T11:00:00.000Z',
  },
]

const tokenUserIdMap = new Map<string, string>()

const projects: ProjectDTO[] = [
  {
    id: 'project_001',
    userId: 'user_001',
    title: '雾港来信',
    novelTitle: '雾港来信',
    sourceText:
      '第一章 雾港的雨下了一整夜。第二章 林晚收到了没有邮戳的来信。第三章 她在码头发现父亲失踪前留下的船票。',
    config: defaultConfig,
    status: 'completed',
    sceneCount: 3,
    errorMessage: null,
    createdAt: '2026-06-06T10:00:00.000Z',
    updatedAt: '2026-06-06T11:00:00.000Z',
  },
  {
    id: 'project_002',
    userId: 'user_001',
    title: '北境药师',
    novelTitle: '雪线之下',
    sourceText:
      '第一章 药师入山。第二章 北境雪线封路。第三章 旧友带来失踪商队的消息。',
    config: {
      ...defaultConfig,
      style: '短剧',
      adaptationMode: '适度改编',
    },
    status: 'completed',
    sceneCount: 48,
    errorMessage: null,
    createdAt: '2026-06-06T09:00:00.000Z',
    updatedAt: '2026-06-06T10:30:00.000Z',
  },
  {
    id: 'project_003',
    userId: 'user_001',
    title: '月台第七封信',
    novelTitle: '夜班列车',
    sourceText: '第一章 深夜月台。第二章 第七封信。第三章 无人认领的行李箱。',
    config: {
      ...defaultConfig,
      style: '短剧',
    },
    status: 'draft',
    sceneCount: 0,
    errorMessage: null,
    createdAt: '2026-06-06T08:20:00.000Z',
    updatedAt: '2026-06-06T08:48:00.000Z',
  },
  {
    id: 'project_004',
    userId: 'user_001',
    title: '青瓷巷口',
    novelTitle: '窑火未眠',
    sourceText: '第一章 青瓷巷口。第二章 老窑重开。第三章 缺口花瓶里的账本。',
    config: defaultConfig,
    status: 'completed',
    sceneCount: 12,
    errorMessage: null,
    createdAt: '2026-06-06T07:50:00.000Z',
    updatedAt: '2026-06-06T08:10:00.000Z',
  },
  {
    id: 'project_005',
    userId: 'user_001',
    title: '白塔灯影',
    novelTitle: '灯塔守夜人',
    sourceText: '第一章 白塔熄灯。第二章 海雾上岸。第三章 守夜人留下航海图。',
    config: {
      ...defaultConfig,
      adaptationMode: '适度改编',
    },
    status: 'generating',
    sceneCount: 0,
    errorMessage: null,
    createdAt: '2026-06-06T07:10:00.000Z',
    updatedAt: '2026-06-06T07:58:00.000Z',
  },
  {
    id: 'project_006',
    userId: 'user_001',
    title: '纸鸢旧事',
    novelTitle: '风从南墙来',
    sourceText: '第一章 纸鸢落在南墙。第二章 旧照片。第三章 失散兄妹重逢。',
    config: defaultConfig,
    status: 'draft',
    sceneCount: 0,
    errorMessage: null,
    createdAt: '2026-06-06T06:40:00.000Z',
    updatedAt: '2026-06-06T07:05:00.000Z',
  },
  {
    id: 'project_007',
    userId: 'user_001',
    title: '榕树下的证词',
    novelTitle: '南城旧案',
    sourceText: '第一章 榕树下。第二章 证词反转。第三章 巷尾照相馆。',
    config: {
      ...defaultConfig,
      style: '影视剧',
    },
    status: 'failed',
    sceneCount: 0,
    errorMessage: '原文结构过短，缺少可拆分的连续情节。',
    createdAt: '2026-06-06T06:00:00.000Z',
    updatedAt: '2026-06-06T06:32:00.000Z',
  },
  {
    id: 'project_008',
    userId: 'user_001',
    title: '雪落第九街',
    novelTitle: '旧城白夜',
    sourceText: '第一章 第九街落雪。第二章 白夜钟声。第三章 失踪画家的钥匙。',
    config: defaultConfig,
    status: 'completed',
    sceneCount: 18,
    errorMessage: null,
    createdAt: '2026-06-05T14:30:00.000Z',
    updatedAt: '2026-06-05T15:20:00.000Z',
  },
  {
    id: 'project_009',
    userId: 'user_001',
    title: '渡口无名船',
    novelTitle: '江雾旧梦',
    sourceText: '第一章 渡口。第二章 无名船。第三章 船舱里的旧报纸。',
    config: {
      ...defaultConfig,
      dialogueLevel: '详细',
    },
    status: 'draft',
    sceneCount: 0,
    errorMessage: null,
    createdAt: '2026-06-05T13:10:00.000Z',
    updatedAt: '2026-06-05T13:44:00.000Z',
  },
  {
    id: 'project_010',
    userId: 'user_001',
    title: '暗房显影',
    novelTitle: '银盐时代',
    sourceText: '第一章 暗房。第二章 显影盘。第三章 被裁掉的人影。',
    config: defaultConfig,
    status: 'completed',
    sceneCount: 9,
    errorMessage: null,
    createdAt: '2026-06-05T12:00:00.000Z',
    updatedAt: '2026-06-05T12:28:00.000Z',
  },
  {
    id: 'project_011',
    userId: 'user_001',
    title: '花窗后的午后',
    novelTitle: '玻璃温室',
    sourceText: '第一章 温室午后。第二章 花窗裂纹。第三章 不该出现的指纹。',
    config: {
      ...defaultConfig,
      narrationLevel: '适中保留',
    },
    status: 'draft',
    sceneCount: 0,
    errorMessage: null,
    createdAt: '2026-06-05T11:00:00.000Z',
    updatedAt: '2026-06-05T11:28:00.000Z',
  },
  {
    id: 'project_012',
    userId: 'user_001',
    title: '长桥尽头',
    novelTitle: '雨季来客',
    sourceText: '第一章 长桥。第二章 雨季来客。第三章 桥尽头的空房间。',
    config: defaultConfig,
    status: 'completed',
    sceneCount: 11,
    errorMessage: null,
    createdAt: '2026-06-05T10:00:00.000Z',
    updatedAt: '2026-06-05T10:35:00.000Z',
  },
]

const createStressTestScenes = (projectId: string) => {
  const locations = ['雪线驿站', '药师木屋', '白桦林', '北境关口', '旧矿洞', '山神庙']
  const timeTexts = ['清晨', '午后', '傍晚', '深夜']
  const characters = ['沈砚', '阿洛', '顾衡', '雪婆婆']
  const createdAt = '2026-06-06T10:20:00.000Z'

  return Array.from({ length: 48 }, (_, index) => {
    const sceneNo = index + 1
    const location = locations[index % locations.length]
    const timeText = timeTexts[index % timeTexts.length]
    const title = `雪线追踪 ${String(sceneNo).padStart(2, '0')}`
    const summary = `沈砚沿着第 ${sceneNo} 条线索推进调查，北境风雪让旧案出现新的破口。`

    return {
      id: `scene_project_002_${String(sceneNo).padStart(3, '0')}`,
      projectId,
      sceneNo,
      title,
      location,
      timeText,
      summary,
      content: `【第 ${sceneNo} 场】${title}\n\n地点：${location}\n时间：${timeText}\n\n动作：风雪压低山脊，沈砚在药箱夹层里发现新的药引记录。\n\n沈砚：这味药不该出现在北境。\n\n阿洛：除非有人早就知道商队会失踪。`,
      rawJson: {
        characters,
        source: {
          chapters: [`第${Math.floor(index / 8) + 1}章`],
          summary: `对应原文中北境调查线的第 ${sceneNo} 个关键节点。`,
        },
      },
      createdAt,
      updatedAt: createdAt,
    } satisfies SceneDTO
  })
}

const createPaginationScenes = (
  projectId: string,
  count: number,
  titlePrefix: string,
  createdAt: string,
) => {
  const locations = ['旧宅书房', '街角茶馆', '雨后长廊', '档案室']
  const timeTexts = ['清晨', '午后', '黄昏', '夜晚']

  return Array.from({ length: count }, (_, index) => {
    const sceneNo = index + 1
    const location = locations[index % locations.length]
    const timeText = timeTexts[index % timeTexts.length]
    const title = `${titlePrefix} ${String(sceneNo).padStart(2, '0')}`

    return {
      id: `scene_${projectId}_${String(sceneNo).padStart(3, '0')}`,
      projectId,
      sceneNo,
      title,
      location,
      timeText,
      summary: `${titlePrefix} 的第 ${sceneNo} 个关键场次，推动人物关系和线索继续展开。`,
      content: `【第 ${sceneNo} 场】${title}\n\n地点：${location}\n时间：${timeText}\n\n动作：人物在${location}重新梳理线索，旧事露出新的缺口。\n\n主角：这不是巧合。\n\n同伴：那就从这里查下去。`,
      rawJson: {
        characters: ['主角', '同伴'],
        source: {
          chapters: [`第${Math.floor(index / 4) + 1}章`],
          summary: `对应 ${titlePrefix} 的阶段性情节点。`,
        },
      },
      createdAt,
      updatedAt: createdAt,
    } satisfies SceneDTO
  })
}

let scenes: SceneDTO[] = [
  {
    id: 'scene_001',
    projectId: 'project_001',
    sceneNo: 1,
    title: '旧书店收到来信',
    location: '旧书店',
    timeText: '雨夜',
    summary: '林晚收到父亲留下的信，失踪七年的谜团重新打开。',
    content:
      '【第 1 场】旧书店收到来信\n\n地点：旧书店\n时间：雨夜\n\n动作：雨水敲打旧书店的玻璃窗。\n\n陈伯：你终于来了。\n\n林晚：这是哪里来的？',
    rawJson: {
      characters: ['林晚', '陈伯'],
      script: [
        { type: 'action', content: '雨水敲打旧书店的玻璃窗。' },
        { type: 'dialogue', character: '陈伯', content: '你终于来了。' },
        { type: 'dialogue', character: '林晚', content: '这是哪里来的？' },
      ],
      source: {
        chapters: ['第一章'],
        summary: '陈伯交出林远山留下的信。',
      },
    },
    createdAt: '2026-06-06T10:10:00.000Z',
    updatedAt: '2026-06-06T10:10:00.000Z',
  },
  {
    id: 'scene_002',
    projectId: 'project_001',
    sceneNo: 2,
    title: '林家老宅重启谜团',
    location: '林家老宅',
    timeText: '清晨',
    summary: '林晚翻出父亲笔记，发现码头编号。',
    content:
      '【第 2 场】林家老宅重启谜团\n\n地点：林家老宅\n时间：清晨\n\n动作：林晚打开父亲的旧箱子，找到一页被雨水晕开的笔记。',
    createdAt: '2026-06-06T10:10:00.000Z',
    updatedAt: '2026-06-06T10:10:00.000Z',
  },
  {
    id: 'scene_003',
    projectId: 'project_001',
    sceneNo: 3,
    title: '雾港码头的旧船票',
    location: '雾港码头',
    timeText: '傍晚',
    summary: '周砚在仓库里承认见过林远山。',
    content:
      '【第 3 场】雾港码头的旧船票\n\n地点：雾港码头\n时间：傍晚\n\n周砚：你父亲最后一次出现，就在这间仓库。',
    createdAt: '2026-06-06T10:10:00.000Z',
    updatedAt: '2026-06-06T10:10:00.000Z',
  },
  ...createStressTestScenes('project_002'),
  ...createPaginationScenes('project_004', 12, '青瓷巷口', '2026-06-06T08:10:00.000Z'),
  ...createPaginationScenes('project_008', 18, '雪落第九街', '2026-06-05T15:20:00.000Z'),
  ...createPaginationScenes('project_010', 9, '暗房显影', '2026-06-05T12:28:00.000Z'),
  ...createPaginationScenes('project_012', 11, '长桥尽头', '2026-06-05T10:35:00.000Z'),
]

let sceneSuggestions: SceneSuggestion[] = []

const readSessionUser = () => {
  const token = localStorage.getItem(authTokenStorageKey)
  if (!token) return null
  const userId = tokenUserIdMap.get(token) ?? token.match(/^mock_token_(user_[^_]+)_/)?.[1]
  return users.find((user) => user.id === userId) ?? users[0]
}

const requireAuth = () => {
  const user = readSessionUser()
  return user ? { user } : { response: fail(40001, '未登录或登录已过期') }
}

const toPublicUser = (user: MockUser): UserDTO => {
  const { password: _password, ...publicUser } = user
  return publicUser
}

const countChapters = (sourceText: string) => {
  return sourceText.match(/第[一二三四五六七八九十百千万\d]+章/g)?.length ?? 0
}

const getProjectForCurrentUser = (projectId: string) => {
  return projects.find((project) => project.id === projectId && project.userId === readSessionUser()?.id)
}

const getSceneForCurrentUser = (sceneId: string) => {
  const scene = scenes.find((item) => item.id === sceneId)
  if (!scene || !getProjectForCurrentUser(scene.projectId)) return null
  return scene
}

const createMockScenes = (projectId: string) => {
  const generatedAt = now()
  return [
    {
      id: createId('scene'),
      projectId,
      sceneNo: 1,
      title: '雨夜来信',
      location: '旧书店',
      timeText: '雨夜',
      summary: '主角收到一封迟到多年的信，故事谜团被打开。',
      content:
        '【第 1 场】雨夜来信\n\n地点：旧书店\n时间：雨夜\n\n动作：雨水敲打玻璃，林晚推门而入。\n\n陈伯：你终于来了。\n\n林晚：这封信，为什么现在才给我？',
      rawJson: {
        characters: ['林晚', '陈伯'],
        script: [
          { type: 'action', content: '雨水敲打玻璃，林晚推门而入。' },
          { type: 'dialogue', character: '陈伯', content: '你终于来了。' },
          { type: 'dialogue', character: '林晚', content: '这封信，为什么现在才给我？' },
        ],
      },
      createdAt: generatedAt,
      updatedAt: generatedAt,
    },
    {
      id: createId('scene'),
      projectId,
      sceneNo: 2,
      title: '旧宅笔记',
      location: '林家老宅',
      timeText: '清晨',
      summary: '林晚从父亲遗物中找到码头编号。',
      content:
        '【第 2 场】旧宅笔记\n\n地点：林家老宅\n时间：清晨\n\n动作：林晚翻开父亲的笔记，纸页边缘已经发脆。\n\n林晚：雾港七码头。',
      createdAt: generatedAt,
      updatedAt: generatedAt,
    },
    {
      id: createId('scene'),
      projectId,
      sceneNo: 3,
      title: '码头旧票',
      location: '雾港码头',
      timeText: '傍晚',
      summary: '旧船票指向父亲最后出现的位置。',
      content:
        '【第 3 场】码头旧票\n\n地点：雾港码头\n时间：傍晚\n\n周砚：你父亲最后一次出现，就在这间仓库。',
      createdAt: generatedAt,
      updatedAt: generatedAt,
    },
  ] satisfies SceneDTO[]
}

const incrementalSceneTemplates = [
  {
    title: '暗房里的第二封信',
    location: '旧照相馆',
    timeText: '夜晚',
    summary: '林晚和周砚在暗房里发现第二封信，父亲失踪前的合影被重新拼上。',
    clue: '被剪掉一角的合影',
    chapter: '第四章',
    characters: ['林晚', '周砚', '许知禾'],
  },
  {
    title: '档案室里的回声',
    location: '市政档案室',
    timeText: '午后',
    summary: '旧档案把雾港码头与林远山的旧案连在一起，新的证人浮出水面。',
    clue: '七号码头封存档案',
    chapter: '第五章',
    characters: ['林晚', '周砚', '档案员'],
  },
  {
    title: '雨棚下的证词',
    location: '码头雨棚',
    timeText: '清晨',
    summary: '守夜人说出当年的最后一班船，林晚确认父亲并非独自离开。',
    clue: '最后一班船的航行记录',
    chapter: '第六章',
    characters: ['林晚', '周砚', '守夜人'],
  },
]

const collectSceneCharacters = (projectScenes: SceneDTO[]) => {
  const characters = projectScenes.flatMap((scene) => [
    ...(scene.rawJson?.characters ?? []),
    ...(scene.rawJson?.script
      ?.map((block) => block.character)
      .filter((character): character is string => Boolean(character)) ?? []),
  ])

  return Array.from(new Set(characters))
}

const createIncrementalMockScenes = (projectId: string, projectScenes: SceneDTO[]) => {
  const generatedAt = now()
  const sortedScenes = [...projectScenes].sort((previous, next) => previous.sceneNo - next.sceneNo)
  const nextSceneNo = (sortedScenes.at(-1)?.sceneNo ?? 0) + 1
  const template = incrementalSceneTemplates[(nextSceneNo - 1) % incrementalSceneTemplates.length]
  const knownCharacters = Array.from(
    new Set([...collectSceneCharacters(sortedScenes), ...template.characters]),
  )
  const lastSceneId = sortedScenes.at(-1)?.id
  const updatedScenes = sortedScenes.map((scene) => {
    if (scene.id !== lastSceneId) return scene

    return {
      ...scene,
      rawJson: {
        ...scene.rawJson,
        characters: Array.from(
          new Set([...(scene.rawJson?.characters ?? []), ...template.characters.slice(0, 2)]),
        ),
        source: {
          ...scene.rawJson?.source,
          chapters: scene.rawJson?.source?.chapters ?? [template.chapter],
          summary: scene.rawJson?.source?.summary
            ? `${scene.rawJson.source.summary}（mock：已同步增量生成后的关系索引。）`
            : 'mock：已同步增量生成后的关系索引。',
        },
      },
      updatedAt: generatedAt,
    } satisfies SceneDTO
  })

  const incrementalScene = {
    id: createId('scene'),
    projectId,
    sceneNo: nextSceneNo,
    title: template.title,
    location: template.location,
    timeText: template.timeText,
    summary: template.summary,
    content: `【第 ${nextSceneNo} 场】${template.title}

地点：${template.location}
时间：${template.timeText}

动作：灯箱亮起，${template.clue}从显影盘边缘慢慢浮出来。

周砚：这不是上一版剧本里能解释的线索。

林晚：那就把它接进来，别让人物关系断掉。

动作：两人把新线索贴回关系墙，旧照片上的空缺终于对上了名字。`,
    rawJson: {
      characters: knownCharacters,
      script: [
        { type: 'action', content: `灯箱亮起，${template.clue}从显影盘边缘慢慢浮出来。` },
        { type: 'dialogue', character: '周砚', content: '这不是上一版剧本里能解释的线索。' },
        { type: 'dialogue', character: '林晚', content: '那就把它接进来，别让人物关系断掉。' },
        { type: 'action', content: '两人把新线索贴回关系墙，旧照片上的空缺终于对上了名字。' },
      ],
      source: {
        chapters: [template.chapter],
        summary: 'mock 增量生成：基于已有剧本补充新场次，并同步当前场景人物与原文依据。',
      },
    },
    createdAt: generatedAt,
    updatedAt: generatedAt,
  } satisfies SceneDTO

  return [...updatedScenes, incrementalScene]
}

const suggestionTemplates = [
  {
    type: 'conflict',
    title: '把隐藏矛盾提前露出一点',
    problem: '当前场次的推进比较顺，角色之间缺少阻力。',
    reason: '观众需要在场景内感到某个秘密正在被靠近，否则这一场容易只承担信息交代。',
    suggestion: '在关键对白前加入一句试探，让对方主动回避或反问，把关系压力显出来。',
    applyText: '周砚：你一直不问那封信是谁送来的，是已经知道答案了吗？',
  },
  {
    type: 'dialogue',
    title: '让对白更像人物在互相试探',
    problem: '对白目前偏直给，人物意图暴露得比较快。',
    reason: '悬疑向场景更适合让角色绕开真实目的，通过短句、停顿和反问制造张力。',
    suggestion: '保留信息点，但把陈述句改成带潜台词的问句或半句。',
    applyText: '林晚：你等的不是我，是我手里的东西，对吗？',
  },
  {
    type: 'rhythm',
    title: '在信息揭示前增加一个动作停顿',
    problem: '线索揭示来得太快，缺少悬念被拉开的瞬间。',
    reason: '动作停顿可以让观众先观察人物反应，再接受关键信息，节奏会更有层次。',
    suggestion: '在揭示信件、船票或旧照片前加入一个具体动作，让角色犹豫半拍。',
    applyText: '动作：林晚伸手去拿信，指尖停在信封边缘，像是怕碰醒什么旧事。',
  },
  {
    type: 'character',
    title: '补足主角的主动选择',
    problem: '当前场次里主角更多是在接收信息。',
    reason: '主角如果在本场做出一个小选择，后续行动会更有因果感。',
    suggestion: '让主角主动提出下一步调查方向，而不是只被线索牵着走。',
    applyText: '林晚：明天一早去码头。我想看看父亲最后站过的地方。',
  },
  {
    type: 'visual',
    title: '增加一个可拍出来的视觉钩子',
    problem: '场景氛围主要依赖文字说明，画面记忆点还不够明确。',
    reason: '剧本场次需要给导演和演员一个可执行的视觉抓手。',
    suggestion: '加入一个贯穿场景的小物件，例如湿信封、旧船票、暗房照片或灯箱。',
    applyText: '动作：雨水从信封角落渗开，露出一枚几乎褪色的码头印章。',
  },
] satisfies Array<Omit<SceneSuggestion, 'id' | 'projectId' | 'sceneId' | 'status' | 'createdAt' | 'updatedAt'>>

const normalizeSuggestionCount = (count?: number) => {
  if (typeof count !== 'number' || !Number.isFinite(count)) return 3
  return Math.min(Math.max(Math.floor(count), 1), 5)
}

const createMockSuggestions = (scene: SceneDTO, content: string, count?: number) => {
  const createdAt = now()
  const suggestionCount = normalizeSuggestionCount(count)
  const sceneCharacters = collectSceneCharacters([scene])
  const primaryCharacter = sceneCharacters[0] ?? '主角'
  const secondaryCharacter = sceneCharacters[1] ?? '对手角色'

  return suggestionTemplates.slice(0, suggestionCount).map((template, index) => {
    const contentHint = content.includes('？') || content.includes('?') ? '疑问句已经存在' : '当前对白较少使用反问'
    const suggestion = {
      ...template,
      id: createId('suggestion'),
      projectId: scene.projectId,
      sceneId: scene.id,
      title: index === 0 ? `${template.title}：${scene.title}` : template.title,
      reason:
        index === 1
          ? `${template.reason}（mock 分析：${contentHint}，可继续强化 ${primaryCharacter} 与 ${secondaryCharacter} 的试探。）`
          : template.reason,
      status: 'pending',
      createdAt,
      updatedAt: createdAt,
    } satisfies SceneSuggestion

    return suggestion
  })
}

const normalizeRegenerationMode = (mode?: string): SceneRegenerationMode => {
  if (mode === 'polish' || mode === 'shorten' || mode === 'expand') return mode
  return 'replace'
}

const createRegeneratedSceneContent = (
  scene: SceneDTO,
  instruction: string,
  mode: SceneRegenerationMode,
) => {
  const characters = collectSceneCharacters([scene])
  const protagonist = characters[0] ?? '主角'
  const counterpart = characters[1] ?? '对手角色'
  const location = scene.location || '当前地点'
  const timeText = scene.timeText || '当前时间'
  const instructionText = instruction || '保留核心剧情，提升对白和动作的可拍性。'

  if (mode === 'shorten') {
    return `【第 ${scene.sceneNo} 场】${scene.title}

地点：${location}
时间：${timeText}

动作：${location}里只剩关键线索，${protagonist}停在门口，迅速判断下一步。

${protagonist}：别绕了，答案就在这里。

${counterpart}：那你最好想清楚，知道以后就回不了头。

动作：${protagonist}收起线索，转身离开。`
  }

  if (mode === 'expand') {
    return `【第 ${scene.sceneNo} 场】${scene.title}

地点：${location}
时间：${timeText}

动作：${location}的空气压得很低，远处的声响被墙面反复弹回。${protagonist}翻开旧物，发现边角处藏着一处此前被忽略的标记。

${counterpart}：你看见了？

${protagonist}：看见了。但我不确定这是提醒，还是警告。

动作：${counterpart}伸手想拿走旧物，${protagonist}先一步按住。

${protagonist}：从现在开始，谁也不能再替我决定真相是什么。

${counterpart}：那就按你的办法来。只是别忘了，你要找的人也可能一直在看着你。

动作：两人对视片刻，灯光忽明忽暗，线索被重新放回桌面中央。`
  }

  if (mode === 'polish') {
    return `【第 ${scene.sceneNo} 场】${scene.title}

地点：${location}
时间：${timeText}

动作：${location}安静下来，${protagonist}把线索摊在桌面上，没有立刻开口。

${counterpart}：你不是来问我的，你是来确认自己已经猜到了。

${protagonist}：如果我猜错了，你现在就该否认。

${counterpart}：我否认，你会信吗？

动作：短暂沉默后，${protagonist}收起线索，眼神比刚才更坚定。`
  }

  return `【第 ${scene.sceneNo} 场】${scene.title}

地点：${location}
时间：${timeText}

动作：${location}里，旧线索被重新摆上桌面。${protagonist}没有急着追问，而是观察${counterpart}的反应。

${protagonist}：你一直知道这件事。

${counterpart}：知道，不代表我有选择。

${protagonist}：那现在呢？你还想继续替别人守着这个秘密？

动作：${counterpart}避开视线，手指压住桌角，纸页轻轻发皱。

${counterpart}：如果我说出来，你要找的答案可能会比失踪更糟。

${protagonist}：我已经走到这里了。

动作：两人之间的沉默被拉长，窗外声响逼近，新的决定在这一刻落下。
${instructionText.includes('动作') ? '\n动作：一个更明确的动作细节压住场尾，给下一场留下入口。' : ''}`
}

Mock.setup({
  timeout: '160-420',
})

Mock.mock(/\/api\/auth\/register$/, 'post', (options: MockRequestOptions) => {
  const payload = parseBody<RegisterRequest>(options)

  if (!payload.email || !payload.password || !payload.nickname) {
    return fail(40002, '参数校验失败')
  }

  if (users.some((user) => user.email === payload.email)) {
    return fail(40003, '邮箱已被注册')
  }

  const createdAt = now()
  const user: MockUser = {
    id: createId('user'),
    email: payload.email,
    password: payload.password,
    nickname: payload.nickname,
    aiPoints: 1000,
    createdAt,
    updatedAt: createdAt,
  }
  users.push(user)

  const token = `mock_token_${user.id}_${Date.now()}`
  tokenUserIdMap.set(token, user.id)
  return ok<AuthTokenDTO>({ token, user: toPublicUser(user) }, '注册成功')
})

Mock.mock(/\/api\/auth\/login$/, 'post', (options: MockRequestOptions) => {
  const payload = parseBody<LoginRequest>(options)

  if (!payload.email || !payload.password) {
    return fail(40002, '参数校验失败')
  }

  const user = users.find(
    (item) => item.email === payload.email && item.password === payload.password,
  )

  if (!user) {
    return fail(40004, '邮箱或密码错误')
  }

  const token = `mock_token_${user.id}_${Date.now()}`
  tokenUserIdMap.set(token, user.id)
  return ok<AuthTokenDTO>({ token, user: toPublicUser(user) }, '登录成功')
})

Mock.mock(/\/api\/auth\/me$/, 'get', () => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response
  return ok(toPublicUser(auth.user))
})

Mock.mock(/\/api\/auth\/logout$/, 'post', () => {
  localStorage.removeItem(authTokenStorageKey)
  return ok(true, '退出登录成功')
})

Mock.mock(/\/api\/projects(?:\?.*)?$/, 'get', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const url = toUrl(options.url)
  const page = Number(url.searchParams.get('page') ?? '1')
  const pageSize = Number(url.searchParams.get('pageSize') ?? '10')
  const keyword = url.searchParams.get('keyword')?.trim()
  const ownProjects = projects
    .filter((project) => project.userId === auth.user.id)
    .filter((project) => {
      if (!keyword) return true
      return project.title.includes(keyword) || project.novelTitle?.includes(keyword)
    })
    .map(({ sourceText: _sourceText, ...project }) => ({
      ...project,
      sceneCount: scenes.filter((scene) => scene.projectId === project.id).length,
    }))

  const start = (page - 1) * pageSize
  const list = ownProjects.slice(start, start + pageSize)

  return ok({
    list,
    total: ownProjects.length,
    page,
    pageSize,
  })
})

Mock.mock(/\/api\/projects$/, 'post', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const payload = parseBody<CreateProjectRequest>(options)

  if (!payload.title || !payload.sourceText || !payload.config) {
    return fail(40002, '参数校验失败')
  }

  if (countChapters(payload.sourceText) < 3) {
    return fail(41001, '小说文本过短')
  }

  const createdAt = now()
  const project: ProjectDTO = {
    id: createId('project'),
    userId: auth.user.id,
    title: payload.title,
    novelTitle: payload.novelTitle,
    sourceText: payload.sourceText,
    config: payload.config,
    status: 'draft',
    sceneCount: 0,
    errorMessage: null,
    createdAt,
    updatedAt: createdAt,
  }
  projects.unshift(project)

  return ok(project, '项目创建成功')
})

Mock.mock(/\/api\/projects\/[^/?]+(?:\?.*)?$/, 'get', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const projectId = getPathSegment(options, 2)
  const project = getProjectForCurrentUser(projectId)

  if (!project) {
    return fail(40401, '项目不存在')
  }

  return ok(project)
})

Mock.mock(/\/api\/projects\/[^/?]+$/, 'patch', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const projectId = getPathSegment(options, 2)
  const project = getProjectForCurrentUser(projectId)

  if (!project) {
    return fail(40401, '项目不存在')
  }

  const payload = parseBody<UpdateProjectRequest>(options)
  if (payload.title !== undefined) project.title = payload.title
  if (payload.novelTitle !== undefined) project.novelTitle = payload.novelTitle
  project.updatedAt = now()

  return ok(
    {
      id: project.id,
      title: project.title,
      novelTitle: project.novelTitle,
      updatedAt: project.updatedAt,
    },
    '项目更新成功',
  )
})

Mock.mock(/\/api\/projects\/[^/?]+$/, 'delete', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const projectId = getPathSegment(options, 2)
  const projectIndex = projects.findIndex(
    (project) => project.id === projectId && project.userId === auth.user.id,
  )

  if (projectIndex < 0) {
    return fail(40401, '项目不存在')
  }

  projects.splice(projectIndex, 1)
  scenes = scenes.filter((scene) => scene.projectId !== projectId)
  sceneSuggestions = sceneSuggestions.filter((suggestion) => suggestion.projectId !== projectId)

  return ok(true, '项目删除成功')
})

Mock.mock(/\/api\/projects\/[^/?]+\/generate$/, 'post', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const projectId = getPathSegment(options, 2)
  const project = getProjectForCurrentUser(projectId)

  if (!project) {
    return fail(40401, '项目不存在')
  }

  if (!project.sourceText) {
    return fail(41002, '项目缺少小说文本')
  }

  if (auth.user.aiPoints < 300) {
    return fail(50001, 'AI 点数不足')
  }

  auth.user.aiPoints -= 300
  auth.user.updatedAt = now()
  project.status = 'completed'
  project.errorMessage = null
  project.updatedAt = now()
  const existingScenes = scenes
    .filter((scene) => scene.projectId === projectId)
    .sort((previous, next) => previous.sceneNo - next.sceneNo)
  scenes = scenes.filter((scene) => scene.projectId !== projectId)
  const generatedScenes =
    existingScenes.length > 0
      ? createIncrementalMockScenes(projectId, existingScenes)
      : createMockScenes(projectId)
  scenes.push(...generatedScenes)
  project.sceneCount = generatedScenes.length

  return ok<GenerateProjectResponse>(
    {
      projectId,
      status: project.status,
      costPoints: 300,
      remainingPoints: auth.user.aiPoints,
      scenes: generatedScenes,
    },
    existingScenes.length > 0 ? '剧本增量生成成功' : '剧本生成成功',
  )
})

Mock.mock(/\/api\/projects\/[^/?]+\/generate\/status$/, 'get', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const projectId = getPathSegment(options, 2)
  const project = getProjectForCurrentUser(projectId)

  if (!project) {
    return fail(40401, '项目不存在')
  }

  const statusMap: Record<string, Pick<GenerateStatusDTO, 'progress' | 'currentStep'>> = {
    draft: { progress: 0, currentStep: '等待生成' },
    generating: { progress: 60, currentStep: '正在拆分剧本场次' },
    completed: { progress: 100, currentStep: '剧本生成完成' },
    failed: { progress: 100, currentStep: project.errorMessage ?? '生成失败' },
  }

  return ok<GenerateStatusDTO>({
    projectId,
    status: project.status,
    ...statusMap[project.status],
  })
})

Mock.mock(/\/api\/projects\/[^/?]+\/scenes(?:\?.*)?$/, 'get', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const projectId = getPathSegment(options, 2)
  const project = getProjectForCurrentUser(projectId)

  if (!project) {
    return fail(40401, '项目不存在')
  }

  return ok(
    scenes
      .filter((scene) => scene.projectId === projectId)
      .sort((previous, next) => previous.sceneNo - next.sceneNo),
  )
})

Mock.mock(/\/api\/scenes\/[^/?]+\/suggestions(?:\?.*)?$/, 'get', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const sceneId = getPathSegment(options, 2)
  const scene = getSceneForCurrentUser(sceneId)

  if (!scene) {
    return fail(40401, '场次不存在')
  }

  const status =
    (toUrl(options.url).searchParams.get('status') as SceneSuggestionStatus | 'all' | null) ??
    'pending'
  const suggestions = sceneSuggestions
    .filter((suggestion) => suggestion.sceneId === sceneId)
    .filter((suggestion) => status === 'all' || suggestion.status === status)
    .sort((previous, next) => Date.parse(next.createdAt) - Date.parse(previous.createdAt))

  return ok(suggestions)
})

Mock.mock(/\/api\/scenes\/[^/?]+\/suggestions$/, 'post', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const sceneId = getPathSegment(options, 2)
  const scene = getSceneForCurrentUser(sceneId)

  if (!scene) {
    return fail(40401, '场次不存在')
  }

  const payload = parseBody<GenerateSceneSuggestionsRequest>(options)
  const content = payload.content?.trim()

  if (!content) {
    return fail(40901, '当前场次内容为空')
  }

  if (auth.user.aiPoints < 30) {
    return fail(40201, 'AI 点数不足，无法生成建议')
  }

  auth.user.aiPoints -= 30
  auth.user.updatedAt = now()

  sceneSuggestions = sceneSuggestions.filter(
    (suggestion) => !(suggestion.sceneId === sceneId && suggestion.status === 'pending'),
  )
  const suggestions = createMockSuggestions(scene, content, payload.count)
  sceneSuggestions.push(...suggestions)

  return ok<GenerateSceneSuggestionsResponse>(
    {
      costPoints: 30,
      remainingPoints: auth.user.aiPoints,
      suggestions,
    },
    'AI 建议生成成功',
  )
})

Mock.mock(/\/api\/scenes\/[^/?]+\/regenerate$/, 'post', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const sceneId = getPathSegment(options, 2)
  const scene = getSceneForCurrentUser(sceneId)

  if (!scene) {
    return fail(40401, '场次不存在')
  }

  const currentContent = scene.content.trim()
  if (!currentContent) {
    return fail(40901, '当前场次内容为空')
  }

  if (auth.user.aiPoints < 80) {
    return fail(40201, 'AI 点数不足，无法重新生成')
  }

  const payload = parseBody<GenerateSceneRegenerationRequest>(options)
  const instruction = payload.instruction?.trim() ?? ''
  const mode = normalizeRegenerationMode(payload.mode)
  const regeneratedContent = createRegeneratedSceneContent(scene, instruction, mode)

  auth.user.aiPoints -= 80
  auth.user.updatedAt = now()

  return ok<GenerateSceneRegenerationResponse>(
    {
      sceneId,
      originalContent: scene.content,
      regeneratedContent,
      instruction,
      costPoints: 80,
      remainingPoints: auth.user.aiPoints,
    },
    '本场重新生成成功',
  )
})

Mock.mock(/\/api\/scenes\/[^/?]+(?:\?.*)?$/, 'get', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const sceneId = getPathSegment(options, 2)
  const scene = scenes.find((item) => item.id === sceneId)

  if (!scene || !getProjectForCurrentUser(scene.projectId)) {
    return fail(40402, '场次不存在')
  }

  return ok(scene)
})

Mock.mock(/\/api\/scenes\/[^/?]+$/, 'patch', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const sceneId = getPathSegment(options, 2)
  const scene = scenes.find((item) => item.id === sceneId)

  if (!scene || !getProjectForCurrentUser(scene.projectId)) {
    return fail(40402, '场次不存在')
  }

  const payload = parseBody<UpdateSceneRequest>(options)

  if (!payload.content) {
    return fail(40002, '参数校验失败')
  }

  scene.title = payload.title ?? scene.title
  scene.location = payload.location ?? scene.location
  scene.timeText = payload.timeText ?? scene.timeText
  scene.summary = payload.summary ?? scene.summary
  scene.content = payload.content
  scene.updatedAt = now()

  return ok<UpdateSceneResponse>(
    {
      id: scene.id,
      updatedAt: scene.updatedAt,
    },
    '场次保存成功',
  )
})

Mock.mock(/\/api\/suggestions\/[^/?]+$/, 'patch', (options: MockRequestOptions) => {
  const auth = requireAuth()
  if ('response' in auth) return auth.response

  const suggestionId = getPathSegment(options, 2)
  const suggestion = sceneSuggestions.find((item) => item.id === suggestionId)

  if (!suggestion || !getProjectForCurrentUser(suggestion.projectId)) {
    return fail(40401, '建议不存在')
  }

  const payload = parseBody<UpdateSceneSuggestionRequest>(options)

  if (payload.status !== 'accepted' && payload.status !== 'dismissed') {
    return fail(40002, '参数校验失败')
  }

  suggestion.status = payload.status
  suggestion.updatedAt = now()

  return ok<UpdateSceneSuggestionResponse>(
    {
      id: suggestion.id,
      status: suggestion.status,
      updatedAt: suggestion.updatedAt,
    },
    '建议状态已更新',
  )
})
