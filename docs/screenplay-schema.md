# 剧本 YAML Schema 设计文档

## 概述

本 Schema 定义了"小说转剧本"工具输出的结构化剧本格式。剧本以 YAML 格式表示，旨在：

1. **人类可读可写** — 作者可以直接编辑 YAML 文件修改剧本
2. **AI 可生成** — LLM 能够根据小说分析结果填充该结构
3. **可渲染** — 前端可以从该结构渲染出标准剧本排版
4. **可验证** — Schema 层次清晰，容易校验完整性

---

## Schema 定义

```yaml
# 根节点：一部完整的剧本
screenplay:
  # 剧本基础信息
  metadata:
    title: string              # 剧本标题
    author: string             # 原作者
    version: string            # 版本号，例如 "1.0"
    created_at: string         # 创建时间，ISO 8601 格式
    original_novel: string     # 原著小说名称
    logline: string            # 一句话梗概
    genre: string[]            # 类型标签，例如 ["悬疑", "推理"]
    source_chapters: int[]     # 来源章节索引，例如 [1, 2, 3, 4, 5]

  # 角色表
  characters:
    - id: string               # 唯一标识符，例如 "char_001"
      name: string             # 角色姓名
      intro: string            # 人物简介（外貌、身份、性格概述）
      gender: string           # 性别（可选）
      age: int                 # 年龄（可选）
      personality: string[]    # 性格关键词，例如 ["冷静", "果断"]
      relationships:           # 人物关系（可选）
        - target: string       # 关联角色 ID
          type: string         # 关系类型，例如 "搭档"、"宿敌"
          description: string  # 关系描述

  # 场景序列
  scenes:
    - id: string               # 唯一标识符，例如 "scene_001"
      sequence: int            # 在剧本中的序号
      heading: string          # 场景标题（Slugline），例如 "内景 刑警支队办公室 - 夜"
      setting:                 # 场景设定
        location: string       # 地点
        time: string           # 时间，例如 "夜"、"1999年夏"
        interior: boolean      # 是否为内景 (true=INT./false=EXT.)
      synopsis: string         # 场景概览（一两句话说明该场景的戏剧目的）
      characters_present: string[]  # 本场景出现的角色 ID 列表
      content:                 # 场景内容序列
        - type: action         # 动作/描写段落
          text: string         # 描述文字（现在时）
        - type: dialogue       # 对白
          character: string    # 说话角色名称（对应 characters 中的 name）
          parenthetical: string  # 表演说明（可选），例如 "(低声)"
          text: string         # 对白内容
        - type: transition     # 转场
          text: string         # 转场标记，例如 "CUT TO:"、"FADE OUT."
        - type: shot           # 镜头指示
          text: string         # 镜头描述，例如 "特写 - 桌上的照片"
        - type: parenthetical  # 独立表演说明（极少使用）
          text: string         # 说明文字
```

---

## 完整示例

```yaml
metadata:
  title: "迷雾追踪"
  author: "佚名"
  version: "1.0"
  created_at: "2026-06-05T10:00:00+08:00"
  original_novel: "迷雾追踪"
  logline: "一名刑警在调查连环失踪案时，发现自己陷入了凶手精心布置的陷阱。"
  genre: ["悬疑", "推理", "犯罪"]
  source_chapters: [1, 2, 3, 4, 5]

characters:
  - id: "char_001"
    name: "林浩"
    intro: "35岁，市刑警支队队长，经验丰富但性格孤僻，离异独居。"
    gender: "男"
    age: 35
    personality: ["敏锐", "固执", "孤独"]
    relationships:
      - target: "char_002"
        type: "搭档"
        description: "多年的刑警搭档，彼此信任但性格互补。"

  - id: "char_002"
    name: "苏晴"
    intro: "28岁，刑警支队技术骨干，犯罪心理学专家，林浩的搭档。"
    gender: "女"
    age: 28
    personality: ["理性", "细致", "坚韧"]
    relationships:
      - target: "char_001"
        type: "搭档"
        description: "与林浩互补的搭档关系，也是唯一能走近他的人。"

  - id: "char_003"
    name: "赵明远"
    intro: "45岁，大学教授，举止儒雅，实际上是一系列失踪案的幕后策划者。"
    gender: "男"
    age: 45
    personality: ["冷静", "高智商", "伪善"]
    relationships:
      - target: "char_001"
        type: "对手"
        description: "将林浩视为唯一的对手，精心设计每一步。"

scenes:
  - id: "scene_001"
    sequence: 1
    heading: "内景 刑警支队办公室 - 夜"
    setting:
      location: "刑警支队办公室"
      time: "夜"
      interior: true
    synopsis: "林浩深夜独自分析案件卷宗，第三个失踪者至今毫无线索，苏晴带来新的物证报告。"
    characters_present: ["char_001", "char_002"]
    content:
      - type: action
        text: "办公室里只亮着一盏台灯。林浩靠在椅背上，面前摊开三份失踪案卷宗。墙上钉满了照片和线索卡片，红线交错如蛛网。窗外是深夜城市的万家灯火。"
      - type: action
        text: "他拿起一张照片——第三个失踪者的面孔——仔细端详，揉了揉太阳穴。"
      - type: dialogue
        character: "林浩"
        text: "三个月，三个人。每次都差一步。"
      - type: action
        text: "门被推开。苏晴端着一杯咖啡走进来，神色中带着一丝兴奋。"
      - type: dialogue
        character: "苏晴"
        parenthetical: "(放下咖啡)"
        text: "技术科出结果了。三个失踪者的手机信号轨迹——你看这个。"
      - type: dialogue
        character: "林浩"
        text: "什么意思？"
      - type: action
        text: "苏晴在桌上摊开一张地图，用红笔圈出三片区域。三个圆圈的交汇点，落在同一个位置。"
      - type: dialogue
        character: "苏晴"
        text: "信号消失前，他们都到过这里。"
      - type: action
        text: "林浩猛地站起来，盯着地图上那个交汇点。"
      - type: dialogue
        character: "林浩"
        text: "城北废弃化工厂……过去十年都没人去过那里。"
      - type: transition
        text: "CUT TO:"

  - id: "scene_002"
    sequence: 2
    heading: "外景 废弃化工厂 - 黎明"
    setting:
      location: "废弃化工厂"
      time: "黎明"
      interior: false
    synopsis: "林浩和苏晴赶到化工厂，发现了重要线索——却也意识到自己已经踏入了陷阱。"
    characters_present: ["char_001", "char_002"]
    content:
      - type: action
        text: "破晓前最暗的时刻。黑色轿车停在生锈的铁门前。林浩下车，手电光束切开铁门内的黑暗。"
      - type: dialogue
        character: "林浩"
        text: "你在车里等着，我先看看。"
      - type: action
        text: "他推开铁门走进去。厂房内部巨大而空旷，头顶破碎的玻璃天窗透进微光。地面上有明显的脚印——新鲜的。"
      - type: action
        text: "林浩蹲下查看脚印。指纹级的细节——不止一个人，而且有拖动重物的痕迹。他顺着痕迹走到一个盖着帆布的物体前。"
      - type: action
        text: "掀开帆布。一个打开的笔记本。上面只有一行字。"
      - type: shot
        text: "特写 - 笔记本上的字"
      - type: parenthetical
        text: "字迹工整，仿佛精心写好等人来读。"
      - type: dialogue
        character: "林浩"
        text: "（读到）"欢迎回来，林队长。我等你很久了。""
      - type: action
        text: "远处传来铁门关闭的声响。林浩猛地回头——铁门已被锁上。他的手机信号一格不剩。"
      - type: dialogue
        character: "林浩"
        text: "（低声）"苏晴——""
      - type: transition
        text: "CUT TO:"
```

---

## 设计原则与原因

### 1. 忠实于行业标准剧本格式

**原则**：Schema 的核心结构参考了 Fountain（纯文本剧本标记语言）和好莱坞标准剧本格式。

**原因**：小说作者虽然熟悉叙事，但不一定了解专业剧本格式。Schema 通过明确的 `heading`（场景标题）、`action`（动作）、`dialogue`（对白）、`transition`（转场）等元素，帮助作者在 AI 辅助下生成符合行业规范的初稿。这是一个"教学性"设计——用户在编辑 YAML 时也在学习剧本结构。

### 2. 三层架构：元数据 → 角色 → 场景

**原则**：顶层分为 `metadata`、`characters`、`scenes` 三个独立区域。

**原因**：
- **`metadata`** 独立出来便于版本管理和检索。`source_chapters` 字段记录了来源章节，这是小说改编特有的需求，方便作者回溯原始材料。
- **`characters`** 先行定义，`scenes` 中的 `characters_present` 和 `content` 中的 `dialogue.character` 都引用角色名称。这种"定义-引用"模式比在每个场景中重复描述角色信息更可控，也方便 AI 在生成时保持角色一致性。
- **`scenes`** 作为核心内容序列，支持 AI 逐步生成（每个场景一次 LLM 调用）或人工逐场修改。

### 3. 场景内容的扁平类型化结构

**原则**：`content` 数组使用 `type` 字段区分动作、对白、转场、镜头指示，而不是使用嵌套对象或 Markdown 混合。

**原因**：
- **可渲染性**：前端可以根据 `type` 选择不同的 CSS/排版样式（动作段落用全宽、对白用居中缩进、角色名用大写等），贴合标准剧本排版。
- **可编辑性**：扁平数组比嵌套结构更易用 JSON Patch 或数组索引操作进行局部编辑。
- **AI 友好**：LLM 生成时只需要关注当前元素类型，不需要管理复杂的嵌套层次。每种类型的字段集最小化（`action` 只需 `text`，`dialogue` 需要 `character` + `text`），降低 LLM 幻觉风险。

### 4. 场景设定与角色关系可选的丰富度

**原则**：`setting` 结构化（location/time/interior）便于检索和分组，`relationships` 支持跨角色关联。

**原因**：
- **跨场景分析**：结构化的 `setting` 让工具可以按地点、时间或内外景进行场景统计和重组，这在小说改编中很有价值（例如合并同一地点的分散场景提高叙事效率）。
- **关系图谱**：`relationships` 为后续功能（AI 辅助生成角色对话、自动检测角色关系变化等）提供基础数据。

### 5. 不改写原文叙事结构

**原则**：Schema 不要求 AI 将小说每一行都翻成剧本格式，而是保留一定程度的叙事段落（`action` 类型可以包含较长的描写）。

**原因**：小说和剧本的本质差异在于"内心独白"和"外部可见动作"的取舍。AI 转换的核心工作是识别哪些小说内容可以/应当转换为对白和动作，哪些需要删除或概括。Schema 的 `action` 段落足够灵活，允许 AI 在初次转换时保留较多的叙述性文字，由作者后续精炼——降低首次转换的门槛。

---

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2026-06-05 | 初始设计 |
