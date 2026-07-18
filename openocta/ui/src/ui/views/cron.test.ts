import { render } from "lit";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CronJob } from "../types.ts";
import { DEFAULT_CRON_FORM } from "../app-defaults.ts";
const nativeConfirmMock = vi.hoisted(() => vi.fn());
vi.mock("../native-dialog-bridge.ts", () => ({
  nativeConfirm: nativeConfirmMock,
}));
import { renderCronConfig, renderCronHistory, type CronProps } from "./cron.ts";

function createJob(id: string): CronJob {
  return {
    id,
    name: "Daily ping",
    enabled: true,
    createdAtMs: 0,
    updatedAtMs: 0,
    schedule: { kind: "cron", expr: "0 9 * * *" },
    sessionTarget: "main",
    wakeMode: "next-heartbeat",
    payload: { kind: "systemEvent", text: "ping" },
  };
}

function createProps(overrides: Partial<CronProps> = {}): CronProps {
  return {
    basePath: "",
    loading: false,
    status: null,
    jobs: [],
    error: null,
    busy: false,
    form: { ...DEFAULT_CRON_FORM },
    addModalOpen: false,
    channels: [],
    channelLabels: {},
    runsJobId: null,
    runs: [],
    onFormChange: () => undefined,
    onRefresh: () => undefined,
    onOpenAddModal: () => undefined,
    onCloseAddModal: () => undefined,
    onAdd: () => undefined,
    onToggle: () => undefined,
    onRun: () => undefined,
    onRemove: () => undefined,
    confirmRemove: false,
    onLoadRuns: () => undefined,
    ...overrides,
  };
}

describe("cron view", () => {
  beforeEach(() => {
    nativeConfirmMock.mockReset();
    document.documentElement.lang = "en";
  });

  it("confirms removal before deleting in scheduled task views", async () => {
    nativeConfirmMock.mockResolvedValueOnce(true);
    const container = document.createElement("div");
    const onRemove = vi.fn();
    render(
      renderCronConfig(
        createProps({
          jobs: [createJob("job-1")],
          confirmRemove: true,
          onRemove,
        }),
      ),
      container,
    );

    const removeButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.trim() === "Remove",
    );
    removeButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await Promise.resolve();

    expect(nativeConfirmMock).toHaveBeenCalledWith("Delete this scheduled task?");
    expect(onRemove).toHaveBeenCalledWith(expect.objectContaining({ id: "job-1" }));
  });

  it("skips removal when scheduled task confirmation is cancelled", async () => {
    nativeConfirmMock.mockResolvedValueOnce(false);
    const container = document.createElement("div");
    const onRemove = vi.fn();
    render(
      renderCronHistory(
        createProps({
          jobs: [createJob("job-1")],
          confirmRemove: true,
          onRemove,
        }),
      ),
      container,
    );

    const removeButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.trim() === "Remove",
    );
    removeButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await Promise.resolve();

    expect(onRemove).not.toHaveBeenCalled();
  });

  it("renders add-job form inside a modal when opened", () => {
    const container = document.createElement("div");
    render(renderCronConfig(createProps({ addModalOpen: true })), container);

    expect(container.querySelector(".cron-config-modal")).not.toBeNull();
    expect(container.querySelector(".emp-detail-modal__close svg")).not.toBeNull();
    expect(container.textContent).toContain("新建任务");
    expect(container.textContent).toContain("Add job");
  });

  it("shows an empty state when there are no jobs to inspect", () => {
    const container = document.createElement("div");
    render(renderCronHistory(createProps()), container);

    expect(container.textContent).toContain("No jobs yet.");
  });

  it("loads run history when selecting a job", () => {
    const container = document.createElement("div");
    const onLoadRuns = vi.fn();
    const job = createJob("job-1");
    render(
      renderCronHistory(
        createProps({
          jobs: [job],
          onLoadRuns,
        }),
      ),
      container,
    );

    const select = container.querySelector("select");
    expect(select).not.toBeNull();
    select!.value = "job-1";
    select?.dispatchEvent(new Event("change", { bubbles: true }));

    expect(onLoadRuns).toHaveBeenCalledWith("job-1");
  });

  it("reflects the selected job in the task selector", () => {
    const container = document.createElement("div");
    const job = createJob("job-1");
    render(
      renderCronHistory(
        createProps({
          jobs: [job],
          runsJobId: "job-1",
        }),
      ),
      container,
    );

    const select = container.querySelector("select") as HTMLSelectElement | null;
    expect(select).not.toBeNull();
    expect(select?.value).toBe("job-1");
  });

  it("renders run chat links when session keys are present", () => {
    const container = document.createElement("div");
    render(
      renderCronHistory(
        createProps({
          basePath: "/ui",
          runsJobId: "job-1",
          runs: [
            {
              ts: Date.now(),
              jobId: "job-1",
              status: "ok",
              summary: "done",
              sessionKey: "agent:main:cron:job-1:run:abc",
            },
          ],
        }),
      ),
      container,
    );

    const link = container.querySelector("a.session-link");
    expect(link).not.toBeNull();
    expect(link?.getAttribute("href")).toContain(
      "/ui/message?session=agent%3Amain%3Acron%3Ajob-1%3Arun%3Aabc",
    );
  });

  it("shows selected job name and sorts run history newest first", () => {
    const container = document.createElement("div");
    const job = createJob("job-1");
    render(
      renderCronHistory(
        createProps({
          jobs: [job],
          runsJobId: "job-1",
          runs: [
            { ts: 1, jobId: "job-1", status: "ok", summary: "older run" },
            { ts: 2, jobId: "job-1", status: "ok", summary: "newer run" },
          ],
        }),
      ),
      container,
    );

    const cards = Array.from(container.querySelectorAll(".card"));
    expect(cards).toHaveLength(1);
    const runHistoryCard = cards[0];

    const select = runHistoryCard.querySelector("select") as HTMLSelectElement | null;
    expect(select?.value).toBe("job-1");

    const summaries = Array.from(
      runHistoryCard?.querySelectorAll(".list-item .list-sub") ?? [],
    ).map((el) => (el.textContent ?? "").trim());
    expect(summaries[0]).toBe("newer run");
    expect(summaries[1]).toBe("older run");
  });
});
