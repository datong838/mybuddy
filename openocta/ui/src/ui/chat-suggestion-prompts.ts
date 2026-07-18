import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { icons } from "./icons.js";

export type ChatSuggestionPrompt = {
  id: string;
  icon: string;
  title: string;
  description: string;
  prompt: string;
};

export type ChatSuggestionCategory = {
  id: string;
  label: string;
  prompts: ChatSuggestionPrompt[];
};

export const CHAT_SUGGESTION_CATEGORIES: ChatSuggestionCategory[] = [
  {
    id: "recommended",
    label: "推荐",
    prompts: [
      {
        id: "host-inspection",
        icon: "🖥️",
        title: "主机服务器巡检",
        description: "采集 CPU、内存、磁盘与服务状态，生成可归档的 Markdown 巡检报告",
        prompt:
          "对目标主机执行服务器巡检并输出 Markdown 报告，包含 CPU、内存、磁盘、关键服务状态、异常项与修复建议。",
      },
      {
        id: "mysql-alert-report",
        icon: "🗄️",
        title: "MySQL 告警分析报告",
        description: "梳理最近 15 分钟 MySQL 告警，输出根因分析与处置优先级",
        prompt: "帮我生成一份最近 15 分钟 MySQL 告警分析报告，包含告警摘要、可能根因、影响范围与处置优先级。",
      },
      {
        id: "list-skills",
        icon: "🧩",
        title: "我能帮你做什么",
        description: "了解当前可用的技能、工具与协同能力，快速找到合适用法",
        prompt: "你能告诉我你有哪些技能吗？请列出已安装的能力以及各自适用场景。",
      },
      {
        id: "local-agent-collab",
        icon: "🤝",
        title: "@Cursor 协同开发",
        description: "委派多文件代码改动给本机 Cursor、Codex 等 CLI Agent",
        prompt:
          "@cursor 在当前工作区实现以下功能：请先帮我整理完整任务描述（含目标、相关文件路径、验收标准），再委派执行。",
      },
      {
        id: "news-daily-brief",
        icon: "📰",
        title: "国际新闻每日简报",
        description: "从 BBC/Reuters 等 RSS 抓取并汇总今日政治、科技、财经要闻",
        prompt:
          "获取今日国际重要新闻（政治、科技、财经），生成中文每日简报，每条附一句话摘要与来源。",
      },
      {
        id: "github-ai-trending",
        icon: "🐙",
        title: "GitHub AI 爆款盘点",
        description: "检索 GitHub 本周 Trending 中 AI/ML 热门项目，生成中文解读报告",
        prompt:
          "检索 GitHub 本周 Trending 中 AI/ML 领域的热门项目，整理成中文报告，包含项目简介、Star 数、适用场景与推荐理由。",
      },
    ],
  },
  {
    id: "office",
    label: "办公学习",
    prompts: [
      {
        id: "meeting-summary",
        icon: "📝",
        title: "会议纪要整理",
        description: "将会议录音或笔记整理为结构化纪要，含待办与负责人",
        prompt: "请将以下会议内容整理为结构化纪要，包含议题、结论、待办事项与负责人。",
      },
      {
        id: "weekly-report",
        icon: "📅",
        title: "周报自动生成",
        description: "根据本周工作记录生成简洁周报，突出成果与风险",
        prompt: "根据我本周的工作记录，生成一份简洁周报，包含完成事项、进行中任务、风险与下周计划。",
      },
      {
        id: "excel-analysis",
        icon: "📊",
        title: "Excel 数据分析",
        description: "分析表格数据，输出关键指标、趋势与可视化建议",
        prompt: "请分析我提供的 Excel 数据，输出关键指标、趋势洞察，并建议合适的图表类型。",
      },
      {
        id: "ontology-memory",
        icon: "🧠",
        title: "项目知识结构化",
        description: "将项目、任务、文档关联为可检索的知识图谱，便于跨会话记忆",
        prompt:
          "记住：我当前主项目是 [项目名]，技术栈为 [技术栈]。请建立与核心模块、待办任务的实体关联，便于后续检索与跟进。",
      },
      {
        id: "local-agents-parallel",
        icon: "⚡",
        title: "多 Agent 并行协作",
        description: "同时委派 Cursor 写实现、Codex 写测试，适合前后端或多模块任务",
        prompt:
          "@cursor 实现 API 接口与业务逻辑；@codex 为同一模块编写 table-driven 单元测试。请先分别整理完整 task 再并行委派。",
      },
      {
        id: "self-improving-review",
        icon: "📋",
        title: "任务复盘与改进",
        description: "复盘刚完成的任务，沉淀错误教训与可复用的最佳实践",
        prompt:
          "对刚才完成的任务做自我复盘：哪些步骤可以做得更好？请将关键教训记录下来，并给出下次同类任务的改进清单。",
      },
    ],
  },
  {
    id: "computer",
    label: "电脑设置",
    prompts: [
      {
        id: "server-patrol-deep",
        icon: "🔍",
        title: "深度主机健康检查",
        description: "全量巡检 CPU、内存、磁盘与服务，输出结构化异常清单",
        prompt:
          "检查目标主机 CPU、内存、磁盘使用情况，列出异常指标并输出结构化 Markdown 巡检报告。",
      },
      {
        id: "host-troubleshoot",
        icon: "🔧",
        title: "主机异常排查",
        description: "梳理 CPU、内存、磁盘异常的排查思路与执行优先级",
        prompt: "帮我梳理主机 CPU、内存、磁盘异常的排查思路，并给出优先级与建议执行的检查命令。",
      },
      {
        id: "desktop-control-win",
        icon: "🪟",
        title: "Windows 桌面自动化",
        description: "窗口切换、进程管理、VSCode 控制与键鼠模拟",
        prompt:
          "列出当前 Windows 占用 CPU/内存最高的 5 个进程，标注是否可安全结束；如需切换窗口或打开 VSCode 文件，请自动处理。",
      },
      {
        id: "disk-cleanup",
        icon: "💾",
        title: "磁盘空间清理",
        description: "分析大文件与可清理目录，给出安全清理建议",
        prompt: "帮我分析本机磁盘占用，找出可安全清理的大文件与目录，并给出分步清理建议。",
      },
      {
        id: "env-setup",
        icon: "⚙️",
        title: "开发环境配置",
        description: "检查并修复常见开发工具链与环境变量问题",
        prompt: "请检查我的开发环境（Node、Go、Python 等），列出缺失组件并给出安装与配置步骤。",
      },
      {
        id: "network-debug",
        icon: "🌐",
        title: "网络连接排查",
        description: "诊断 DNS、代理与端口连通性问题",
        prompt: "我遇到网络连接问题，请帮我系统排查 DNS、代理、防火墙与目标端口连通性。",
      },
    ],
  },
  {
    id: "daily",
    label: "生活日常",
    prompts: [
      {
        id: "agent-stock-screen",
        icon: "📈",
        title: "短线交易选股",
        description: "基于量化规则筛选 A 股短线候选标的，附入选理由与风险提示",
        prompt:
          "帮我完成一次短线选股：说明筛选条件、输出候选个股列表，并解释每只的入选理由与风险提示。",
      },
      {
        id: "agent-stock-holdings",
        icon: "💼",
        title: "持仓分析与调仓",
        description: "从风险控制、行业集中度角度分析持仓并给出调仓建议",
        prompt: "请分析我当前的持仓情况，从风险控制、行业集中度、止损止盈角度给出调仓建议与优先级。",
      },
      {
        id: "news-tech-voice",
        icon: "🔊",
        title: "科技新闻语音简报",
        description: "抓取科技 RSS 热点，生成适合 3 分钟播报的口语化中文稿",
        prompt:
          "从 BBC/Reuters 科技 RSS 获取今日 Top 10 科技新闻，生成适合 3 分钟语音播报的中文稿（分条、口语化）。",
      },
      {
        id: "travel-plan",
        icon: "✈️",
        title: "旅行行程规划",
        description: "根据目的地与天数生成详细行程与预算估算",
        prompt: "请帮我规划一次 3 天的旅行行程，包含每日路线、餐饮推荐与大致预算。",
      },
      {
        id: "recipe",
        icon: "🍳",
        title: "今日菜谱推荐",
        description: "根据现有食材推荐简单可行的家常菜",
        prompt: "我有以下食材，请推荐 2-3 道简单家常菜，附步骤与预计用时。",
      },
      {
        id: "budget",
        icon: "💰",
        title: "月度预算分析",
        description: "分类汇总支出并给出节省建议",
        prompt: "请帮我分析本月支出分类，找出可优化项，并给出下月预算建议。",
      },
    ],
  },
  {
    id: "entertainment",
    label: "游戏娱乐",
    prompts: [
      {
        id: "remotion-video",
        icon: "🎬",
        title: "数据驱动短视频",
        description: "用 React + Remotion 程序化生成带图表动画的 MP4 视频模板",
        prompt:
          "帮我设计一个数据可视化短视频：输入 JSON 业务指标，生成带动画图表的视频脚本、场景结构与 React 组件大纲。",
      },
      {
        id: "game-guide",
        icon: "🎮",
        title: "游戏攻略整理",
        description: "汇总主线流程、隐藏要素与新手避坑指南",
        prompt: "请整理这款游戏的入门攻略，包含主线流程、资源优先级与新手常见误区。",
      },
      {
        id: "movie-rec",
        icon: "🎞️",
        title: "影视推荐",
        description: "根据口味偏好推荐片单，附一句话推荐理由",
        prompt: "我喜欢科幻与悬疑类电影，请推荐 8 部高质量片单，每部附一句话推荐理由。",
      },
      {
        id: "music-playlist",
        icon: "🎵",
        title: "歌单生成",
        description: "按场景与 mood 生成 20 首左右的播放列表",
        prompt: "请生成一份适合专注工作的歌单，约 20 首，风格以轻电子与纯音乐为主。",
      },
      {
        id: "story-idea",
        icon: "📖",
        title: "故事灵感",
        description: "根据关键词生成短故事大纲与人物设定",
        prompt: "请以「时间旅行 + 咖啡店」为关键词，生成一个短故事大纲与主要人物设定。",
      },
      {
        id: "meme-caption",
        icon: "😄",
        title: "段子与文案",
        description: "为社交媒体生成轻松幽默的短文案",
        prompt: "请为周末放松主题写 5 条适合发朋友圈的轻松幽默短文案。",
      },
    ],
  },
];

function promptsFromStrings(items: string[]): ChatSuggestionPrompt[] {
  return items.map((prompt, index) => ({
    id: `custom-${index}`,
    icon: "💬",
    title: prompt.length > 18 ? `${prompt.slice(0, 17)}…` : prompt,
    description: prompt,
    prompt,
  }));
}

function resolveCategories(extraPrompts?: string[]): ChatSuggestionCategory[] {
  if (!extraPrompts?.length) {
    return CHAT_SUGGESTION_CATEGORIES;
  }
  const [first, ...rest] = CHAT_SUGGESTION_CATEGORIES;
  return [
    {
      ...first,
      prompts: [...promptsFromStrings(extraPrompts), ...first.prompts].slice(0, 6),
    },
    ...rest,
  ];
}

@customElement("openocta-chat-suggestions")
export class OpenOctaChatSuggestions extends LitElement {
  @property({ type: Boolean }) disabled = false;
  @property({ attribute: false }) extraPrompts?: string[];
  @state() private activeCategoryId = "recommended";

  protected createRenderRoot() {
    return this;
  }

  private get categories() {
    return resolveCategories(this.extraPrompts);
  }

  private get activeCategory() {
    return this.categories.find((c) => c.id === this.activeCategoryId) ?? this.categories[0];
  }

  private selectPrompt(prompt: string) {
    this.dispatchEvent(
      new CustomEvent("suggestion-select", {
        detail: { prompt },
        bubbles: true,
        composed: true,
      }),
    );
  }

  render() {
    const category = this.activeCategory;
    if (!category) {
      return nothing;
    }

    return html`
      <div class="chat-suggestions">
        <div class="chat-suggestions__tabs" role="tablist" aria-label="场景分类">
          ${this.categories.map(
            (item) => html`
              <button
                type="button"
                class="chat-suggestions__tab ${item.id === category.id ? "chat-suggestions__tab--active" : ""}"
                role="tab"
                aria-selected=${item.id === category.id ? "true" : "false"}
                ?disabled=${this.disabled}
                @click=${() => {
                  this.activeCategoryId = item.id;
                }}
              >
                ${item.label}
              </button>
            `,
          )}
        </div>
        <div class="chat-suggestions__grid">
          ${category.prompts.map(
            (item) => html`
              <button
                type="button"
                class="chat-suggestions__card"
                ?disabled=${this.disabled}
                @click=${() => this.selectPrompt(item.prompt)}
              >
                <span class="chat-suggestions__card-icon" aria-hidden="true">${item.icon}</span>
                <span class="chat-suggestions__card-body">
                  <span class="chat-suggestions__card-title">${item.title}</span>
                  <span class="chat-suggestions__card-desc">${item.description}</span>
                </span>
                <span class="chat-suggestions__card-action" aria-hidden="true">${icons.arrowUpRight}</span>
              </button>
            `,
          )}
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "openocta-chat-suggestions": OpenOctaChatSuggestions;
  }
}
