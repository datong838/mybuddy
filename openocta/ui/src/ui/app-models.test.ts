import { beforeEach, describe, expect, it, vi } from "vitest";
const nativeConfirmMock = vi.hoisted(() => vi.fn());
const saveConfigPatchMock = vi.hoisted(() => vi.fn());

vi.mock("./native-dialog-bridge.ts", () => ({
  nativeConfirm: nativeConfirmMock,
}));

vi.mock("./controllers/config.ts", async () => {
  const actual = await vi.importActual<typeof import("./controllers/config.ts")>("./controllers/config.ts");
  return {
    ...actual,
    saveConfigPatch: saveConfigPatchMock,
  };
});

import type { AppViewState } from "./app-view-state.ts";
import { handleModelsDeleteProvider, handleModelsRemoveModel, handleModelsSave } from "./app-models.ts";

describe("handleModelsDeleteProvider", () => {
  beforeEach(() => {
    nativeConfirmMock.mockReset();
    saveConfigPatchMock.mockReset();
  });

  it("sends null patches for deleted providers and related model env", async () => {
    nativeConfirmMock.mockResolvedValue(true);
    saveConfigPatchMock.mockResolvedValue(true);

    const state = {
      modelsSelectedProvider: "custom-provider",
      modelLibrarySelectedProvider: "custom-provider",
      configForm: {
        models: {
          providers: {
            "custom-provider": {
              displayName: "Custom Provider",
              models: [
                { id: "alpha", name: "Alpha" },
                { id: "beta", name: "Beta" },
              ],
            },
            another: {
              displayName: "Another Provider",
              models: [{ id: "gamma", name: "Gamma" }],
            },
          },
        },
        env: {
          modelEnv: {
            "custom-provider/alpha": { API_KEY: "$ALPHA_KEY" },
            "custom-provider/beta": { API_KEY: "$BETA_KEY" },
            "another/gamma": { API_KEY: "$GAMMA_KEY" },
          },
        },
        agents: {
          defaults: {
            model: {
              primary: "custom-provider/alpha",
            },
          },
        },
      },
      configSnapshot: null,
      configFormDirty: false,
      modelsFormDirty: false,
      client: {} as AppViewState["client"],
      connected: true,
      lastError: null,
    } as unknown as AppViewState;

    await handleModelsDeleteProvider(state);

    expect(saveConfigPatchMock).toHaveBeenCalledTimes(1);
    expect(saveConfigPatchMock).toHaveBeenCalledWith(state, {
      models: {
        providers: {
          "custom-provider": null,
        },
      },
      env: {
        modelEnv: {
          "custom-provider/alpha": null,
          "custom-provider/beta": null,
        },
      },
      agents: {
        defaults: {
          model: {
            primary: null,
          },
        },
      },
    });
    expect(state.modelsSelectedProvider).toBeNull();
    expect(state.modelLibrarySelectedProvider).toBeNull();
  });

  it("keeps the provider selected when delete save fails", async () => {
    nativeConfirmMock.mockResolvedValue(true);
    saveConfigPatchMock.mockResolvedValue(false);

    const state = {
      modelsSelectedProvider: "custom-provider",
      modelLibrarySelectedProvider: "custom-provider",
      configForm: {
        models: {
          providers: {
            "custom-provider": {
              displayName: "Custom Provider",
              models: [{ id: "alpha", name: "Alpha" }],
            },
          },
        },
      },
      configSnapshot: null,
      configFormDirty: false,
      modelsFormDirty: false,
      client: {} as AppViewState["client"],
      connected: true,
      lastError: null,
    } as unknown as AppViewState;

    saveConfigPatchMock.mockImplementation(async (hostArg) => {
      hostArg.lastError = "save failed";
    });
    await handleModelsDeleteProvider(state);

    expect(saveConfigPatchMock).toHaveBeenCalledTimes(1);
    expect(state.modelsSelectedProvider).toBe("custom-provider");
    expect(state.modelLibrarySelectedProvider).toBe("custom-provider");
  });

  it("stops when deletion is not confirmed", async () => {
    nativeConfirmMock.mockResolvedValue(false);
    saveConfigPatchMock.mockResolvedValue(true);

    const state = {
      modelsSelectedProvider: "custom-provider",
      modelLibrarySelectedProvider: "custom-provider",
      configForm: {
        models: {
          providers: {
            "custom-provider": {
              displayName: "Custom Provider",
              models: [{ id: "alpha", name: "Alpha" }],
            },
          },
        },
      },
      configSnapshot: null,
      configFormDirty: false,
      modelsFormDirty: false,
      client: {} as AppViewState["client"],
      connected: true,
      lastError: null,
    } as unknown as AppViewState;

    await handleModelsDeleteProvider(state);

    expect(saveConfigPatchMock).not.toHaveBeenCalled();
    expect(state.modelsSelectedProvider).toBe("custom-provider");
  });

  it("removes the model env entry from form state when a model is deleted", () => {
    const state = {
      configForm: {
        models: {
          providers: {
            "custom-provider": {
              displayName: "Custom Provider",
              models: [
                { id: "alpha", name: "Alpha" },
                { id: "beta", name: "Beta" },
              ],
            },
          },
        },
        env: {
          modelEnv: {
            "custom-provider/alpha": { API_KEY: "$ALPHA_KEY" },
            "custom-provider/beta": { API_KEY: "$BETA_KEY" },
          },
        },
        agents: {
          defaults: {
            model: {
              primary: "custom-provider/alpha",
            },
          },
        },
      },
      configSnapshot: null,
      configFormDirty: false,
      modelsFormDirty: false,
    } as unknown as AppViewState;

    handleModelsRemoveModel(state, "custom-provider", "alpha");

    expect((state.configForm as Record<string, any>).env.modelEnv).toEqual({
      "custom-provider/beta": { API_KEY: "$BETA_KEY" },
    });
    expect((state.configForm as Record<string, any>).agents.defaults.model.primary).toBeUndefined();
  });

  it("marks missing model env entries as null when saving", () => {
    saveConfigPatchMock.mockResolvedValue(true);

    const state = {
      configForm: {
        models: {
          providers: {
            "custom-provider": {
              displayName: "Custom Provider",
              models: [
                { id: "alpha", name: "Alpha" },
                { id: "beta", name: "Beta" },
              ],
            },
          },
        },
        env: {
          vars: {},
          modelEnv: {
            "custom-provider/alpha": { API_KEY: "$ALPHA_KEY" },
            "custom-provider/beta": { API_KEY: "$BETA_KEY" },
          },
        },
        agents: {
          defaults: {
            model: {
              primary: "custom-provider/alpha",
            },
          },
        },
      },
      configFormOriginal: {
        env: {
          modelEnv: {
            "custom-provider/alpha": { API_KEY: "$ALPHA_KEY" },
            "custom-provider/beta": { API_KEY: "$BETA_KEY" },
          },
        },
      },
      configSnapshot: null,
      configFormDirty: false,
      modelsFormDirty: false,
      modelsSaveError: null,
      modelsSelectedProvider: "custom-provider",
      modelLibrarySelectedProvider: "custom-provider",
    } as unknown as AppViewState;

    handleModelsRemoveModel(state, "custom-provider", "alpha");
    handleModelsSave(state);

    expect(saveConfigPatchMock).toHaveBeenCalledTimes(1);
    expect(saveConfigPatchMock.mock.calls[0]?.[1]).toEqual(expect.objectContaining({
      env: {
        vars: {},
        modelEnv: {
          "custom-provider/alpha": null,
          "custom-provider/beta": { API_KEY: "$BETA_KEY" },
        },
      },
      agents: {
        defaults: {
          model: {},
        },
      },
    }));
    expect(state.modelsSelectedProvider).toBeNull();
    expect(state.modelLibrarySelectedProvider).toBeNull();
  });

  it("updates local config when provider deletion starts while disconnected", async () => {
    nativeConfirmMock.mockResolvedValue(true);

    const state = {
      modelsSelectedProvider: "custom-provider",
      modelLibrarySelectedProvider: "custom-provider",
      configForm: {
        models: {
          providers: {
            "custom-provider": {
              displayName: "Custom Provider",
              models: [{ id: "alpha", name: "Alpha" }],
            },
          },
        },
        env: {
          modelEnv: {
            "custom-provider/alpha": { API_KEY: "$ALPHA_KEY" },
          },
        },
      },
      configSnapshot: null,
      configFormDirty: false,
      modelsFormDirty: false,
      client: null,
      connected: false,
      lastError: null,
    } as unknown as AppViewState;

    await handleModelsDeleteProvider(state);

    expect(saveConfigPatchMock).not.toHaveBeenCalled();
    expect(nativeConfirmMock).toHaveBeenCalledTimes(1);
    expect((state.configForm as Record<string, unknown>).models).toEqual({ providers: {} });
    expect((state.configForm as Record<string, any>).env.modelEnv).toEqual({});
    expect(state.modelsSelectedProvider).toBeNull();
    expect(state.modelLibrarySelectedProvider).toBeNull();
    expect(state.configFormDirty).toBe(true);
    expect(state.modelsFormDirty).toBe(true);
  });

  it("deletes provider using model library selection when models tab selection is empty", async () => {
    nativeConfirmMock.mockResolvedValue(true);
    saveConfigPatchMock.mockResolvedValue(true);

    const state = {
      modelsSelectedProvider: null,
      modelLibrarySelectedProvider: "custom-provider",
      configForm: {
        models: {
          providers: {
            "custom-provider": {
              displayName: "Custom Provider",
              models: [{ id: "alpha", name: "Alpha" }],
            },
          },
        },
      },
      configSnapshot: null,
      configFormDirty: false,
      modelsFormDirty: false,
      client: {} as AppViewState["client"],
      connected: true,
      lastError: null,
    } as unknown as AppViewState;

    await handleModelsDeleteProvider(state);

    expect(saveConfigPatchMock).toHaveBeenCalledWith(state, {
      models: {
        providers: {
          "custom-provider": null,
        },
      },
      env: {
        modelEnv: {
          "custom-provider/alpha": null,
        },
      },
    });
    expect(state.modelLibrarySelectedProvider).toBeNull();
  });
});
