import { render } from "lit";
import { describe, expect, it, vi } from "vitest";
import { renderAgents } from "./agents.ts";

describe("agents view", () => {
  it("renders fallback model labels without plus-prefixed counts", () => {
    const container = document.createElement("div");
    render(
      renderAgents({
        loading: false,
        error: null,
        agentsList: {
          defaultId: "agent-1",
          mainKey: "agent-1",
          scope: "workspace",
          agents: [{ id: "agent-1", name: "Agent One" }],
        },
        selectedAgentId: "agent-1",
        activePanel: "overview",
        configForm: {
          agents: {
            list: [
              {
                id: "agent-1",
                model: {
                  primary: "provider/model-a",
                  fallbacks: ["provider/model-b"],
                },
              },
            ],
            defaults: {},
          },
        },
        configLoading: false,
        configSaving: false,
        configDirty: false,
        channelsLoading: false,
        channelsError: null,
        channelsSnapshot: null,
        channelsLastSuccess: null,
        cronLoading: false,
        cronStatus: null,
        cronJobs: [],
        cronError: null,
        agentFilesLoading: false,
        agentFilesError: null,
        agentFilesList: null,
        agentFileActive: null,
        agentFileContents: {},
        agentFileDrafts: {},
        agentFileSaving: false,
        agentIdentityLoading: false,
        agentIdentityError: null,
        agentIdentityById: {},
        agentSkillsLoading: false,
        agentSkillsReport: null,
        agentSkillsError: null,
        agentSkillsAgentId: null,
        skillsFilter: "",
        onRefresh: vi.fn(),
        onSelectAgent: vi.fn(),
        onSelectPanel: vi.fn(),
        onLoadFiles: vi.fn(),
        onSelectFile: vi.fn(),
        onFileDraftChange: vi.fn(),
        onFileReset: vi.fn(),
        onFileSave: vi.fn(),
        onToolsProfileChange: vi.fn(),
        onToolsOverridesChange: vi.fn(),
        onConfigReload: vi.fn(),
        onConfigSave: vi.fn(),
        onModelChange: vi.fn(),
        onModelFallbacksChange: vi.fn(),
        onChannelsRefresh: vi.fn(),
        onCronRefresh: vi.fn(),
        onSkillsFilterChange: vi.fn(),
        onSkillsRefresh: vi.fn(),
        onAgentSkillToggle: vi.fn(),
        onAgentSkillsClear: vi.fn(),
        onAgentSkillsDisableAll: vi.fn(),
      }),
      container,
    );

    expect(container.textContent).toContain("provider/model-a (1 fallback)");
    expect(container.textContent).not.toContain("(+1 fallback)");
  });
});
