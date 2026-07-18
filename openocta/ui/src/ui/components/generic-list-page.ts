import { html, type TemplateResult } from "lit";
import { AsyncDirective } from "lit/async-directive.js";
import { directive, type PartInfo, PartType } from "lit/directive.js";
import { repeat } from "lit/directives/repeat.js";

let genericListPageId = 0;

export type GenericListPageOptions<T> = {
  items: T[];
  renderItem: (item: T, index: number) => TemplateResult;
  keyFn?: (item: T, index: number) => unknown;
  initialCount?: number;
  batchSize?: number;
  containerClass?: string;
  sentinelLabel?: string;
  disabled?: boolean;
};

export type GenericListSection<T, S extends Record<string, unknown> = Record<string, unknown>> = S & {
  key: unknown;
  title: unknown;
  items: T[];
  sectionClass?: string;
  gridClass?: string;
};

export type GenericSectionListPageOptions<
  T,
  S extends GenericListSection<T> = GenericListSection<T>,
> = {
  sections: S[];
  renderItem: (item: T, index: number, section: S) => TemplateResult;
  keyFn?: (item: T, index: number, section: S) => unknown;
  initialCount?: number;
  batchSize?: number;
  sectionsClass?: string;
  sectionClass?: string;
  gridClass?: string;
  sentinelLabel?: string;
  disabled?: boolean;
};

function findScrollRoot(element: HTMLElement) {
  let current = element.parentElement;
  while (current && current !== document.body) {
    const style = window.getComputedStyle(current);
    const overflowY = style.overflowY;
    if (overflowY === "auto" || overflowY === "scroll") {
      return current;
    }
    current = current.parentElement;
  }
  return null;
}

class GenericListPageDirective<T> extends AsyncDirective {
  private readonly sentinelId = `generic-list-sentinel-${++genericListPageId}`;
  private visibleCount = 0;
  private observer?: IntersectionObserver;
  private keys = "";
  private options?: GenericListPageOptions<T>;

  constructor(partInfo: PartInfo) {
    super(partInfo);
    if (partInfo.type !== PartType.CHILD) {
      throw new Error("genericListPage can only be used in child expressions");
    }
  }

  disconnected() {
    this.observer?.disconnect();
    this.observer = undefined;
  }

  reconnected() {
    this.scheduleObserve();
  }

  render(options: GenericListPageOptions<T>) {
    this.options = options;
    const items = options.items ?? [];
    const keyFn = options.keyFn ?? ((_item: T, index: number) => index);
    const initialCount = options.initialCount ?? 24;
    const nextKeys = items.map((item, index) => String(keyFn(item, index))).join("");

    if (nextKeys !== this.keys) {
      this.keys = nextKeys;
      this.visibleCount = initialCount;
    } else if (this.visibleCount === 0) {
      this.visibleCount = initialCount;
    }

    const visibleItems = options.disabled ? items : items.slice(0, this.visibleCount);
    const hasMore = !options.disabled && this.visibleCount < items.length;

    this.scheduleObserve();

    return html`
      <div class=${options.containerClass ?? ""}>
        ${repeat(
          visibleItems,
          (item, index) => keyFn(item, index),
          (item, index) => options.renderItem(item, index),
        )}
      </div>
      ${hasMore
        ? html`
            <div
              id=${this.sentinelId}
              class="generic-list-sentinel"
              role="status"
              aria-label=${`${options.sentinelLabel ?? "继续加载"}，已显示 ${visibleItems.length} / ${items.length}`}
            >
              <span>${options.sentinelLabel ?? "继续加载"}</span>
            </div>
          `
        : ""}
    `;
  }

  update(_part: unknown, [options]: [GenericListPageOptions<T>]) {
    return this.render(options);
  }

  private scheduleObserve() {
    queueMicrotask(() => this.observeSentinel());
  }

  private observeSentinel() {
    const sentinel = document.getElementById(this.sentinelId);

    if (!sentinel || !this.options || this.visibleCount >= (this.options.items?.length ?? 0)) {
      this.observer?.disconnect();
      this.observer = undefined;
      return;
    }

    this.observer?.disconnect();
    this.observer = new IntersectionObserver(
      (entries) => {
        if (!entries.some((entry) => entry.isIntersecting) || !this.options) return;
        this.visibleCount = Math.min(
          this.options.items.length,
          this.visibleCount + (this.options.batchSize ?? 24),
        );
        this.setValue(this.render(this.options));
      },
      { root: findScrollRoot(sentinel), rootMargin: "600px 0px", threshold: 0.01 },
    );
    this.observer.observe(sentinel);
  }
}

class GenericSectionListPageDirective<
  T,
  S extends GenericListSection<T> = GenericListSection<T>,
> extends AsyncDirective {
  private readonly sentinelId = `generic-section-list-sentinel-${++genericListPageId}`;
  private visibleCount = 0;
  private observer?: IntersectionObserver;
  private keys = "";
  private options?: GenericSectionListPageOptions<T, S>;

  constructor(partInfo: PartInfo) {
    super(partInfo);
    if (partInfo.type !== PartType.CHILD) {
      throw new Error("genericSectionListPage can only be used in child expressions");
    }
  }

  disconnected() {
    this.observer?.disconnect();
    this.observer = undefined;
  }

  reconnected() {
    this.scheduleObserve();
  }

  render(options: GenericSectionListPageOptions<T, S>) {
    this.options = options;
    const sections = options.sections ?? [];
    const keyFn = options.keyFn ?? ((item: T, index: number) => index);
    const initialCount = options.initialCount ?? 96;
    const flatKeys = sections.flatMap((section) => [
      String(section.key),
      ...section.items.map((item, index) => String(keyFn(item, index, section))),
    ]);
    const nextKeys = flatKeys.join("");

    if (nextKeys !== this.keys) {
      this.keys = nextKeys;
      this.visibleCount = initialCount;
    } else if (this.visibleCount === 0) {
      this.visibleCount = initialCount;
    }

    const totalCount = sections.reduce((sum, section) => sum + section.items.length, 0);
    const hasMore = !options.disabled && this.visibleCount < totalCount;
    let remaining = options.disabled ? totalCount : this.visibleCount;

    this.scheduleObserve();

    return html`
      <div class=${options.sectionsClass ?? ""}>
        ${sections.map((section) => {
          if (remaining <= 0) return "";
          const visibleItems = section.items.slice(0, remaining);
          remaining -= visibleItems.length;
          if (visibleItems.length === 0) return "";

          return html`
            <div class=${section.sectionClass ?? options.sectionClass ?? ""}>
              <div class="emp-section__header">
                <h3 class="emp-section__title">${section.title}</h3>
              </div>
              <div class=${section.gridClass ?? options.gridClass ?? ""}>
                ${repeat(
                  visibleItems,
                  (item, index) => keyFn(item, index, section),
                  (item, index) => options.renderItem(item, index, section),
                )}
              </div>
            </div>
          `;
        })}
      </div>
      ${hasMore
        ? html`
            <div
              id=${this.sentinelId}
              class="generic-list-sentinel"
              role="status"
              aria-label=${`${options.sentinelLabel ?? "继续加载"}，已显示 ${Math.min(this.visibleCount, totalCount)} / ${totalCount}`}
            >
              <span>${options.sentinelLabel ?? "继续加载"}</span>
            </div>
          `
        : ""}
    `;
  }

  update(_part: unknown, [options]: [GenericSectionListPageOptions<T, S>]) {
    return this.render(options);
  }

  private scheduleObserve() {
    queueMicrotask(() => this.observeSentinel());
  }

  private observeSentinel() {
    const sentinel = document.getElementById(this.sentinelId);
    const totalCount = this.options?.sections.reduce((sum, section) => sum + section.items.length, 0) ?? 0;

    if (!sentinel || !this.options || this.visibleCount >= totalCount) {
      this.observer?.disconnect();
      this.observer = undefined;
      return;
    }

    this.observer?.disconnect();
    this.observer = new IntersectionObserver(
      (entries) => {
        if (!entries.some((entry) => entry.isIntersecting) || !this.options) return;
        this.visibleCount = Math.min(totalCount, this.visibleCount + (this.options.batchSize ?? 96));
        this.setValue(this.render(this.options));
      },
      { root: findScrollRoot(sentinel), rootMargin: "600px 0px", threshold: 0.01 },
    );
    this.observer.observe(sentinel);
  }
}

const genericListPageDirective = directive(GenericListPageDirective);
const genericSectionListPageDirective = directive(GenericSectionListPageDirective);

export function genericListPage<T>(options: GenericListPageOptions<T>) {
  return genericListPageDirective(options as GenericListPageOptions<unknown>);
}

export function genericSectionListPage<
  T,
  S extends GenericListSection<T> = GenericListSection<T>,
>(options: GenericSectionListPageOptions<T, S>) {
  return genericSectionListPageDirective(
    options as GenericSectionListPageOptions<unknown, GenericListSection<unknown>>,
  );
}
