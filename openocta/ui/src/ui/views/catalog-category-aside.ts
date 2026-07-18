import { html, type TemplateResult } from "lit";
import type { McpListItem, SkillListItem } from "../controllers/remote-market.ts";
import type { ModelProvider } from "./models.ts";
import { computeModelLibraryCategories, type ModelLibraryCategory } from "./model-library.ts";

export function renderCatalogCategoryNav(content: TemplateResult) {
  return html`
    <div class="nav-group">
      <div class="nav-group__items">
        ${content}
      </div>
    </div>
  `;
}

export function renderSkillCategoryNav(props: {
  items: SkillListItem[];
  selectedCategory: string;
  keyword: string;
  gatewayHost?: string;
  token?: string;
  reloadVersion: number;
  disabled: boolean;
  onSelect: (name: string, descendantNames: string[]) => void;
}) {
  return renderCatalogCategoryNav(html`
    <category-tree-sidebar
      scope="skill"
      .items=${props.items}
      .selectedCategory=${props.selectedCategory || "__all__"}
      .keyword=${props.keyword}
      .gatewayHost=${props.gatewayHost ?? ""}
      .token=${props.token ?? ""}
      .reloadVersion=${props.reloadVersion}
      ?disabled=${props.disabled}
      @category-select=${(e: CustomEvent<{ name: string; descendantNames?: string[] }>) => {
        props.onSelect(e.detail.name, e.detail.descendantNames ?? []);
      }}
    ></category-tree-sidebar>
  `);
}

export function renderToolCategoryNav(props: {
  items: McpListItem[];
  selectedCategory: string;
  keyword: string;
  gatewayHost?: string;
  token?: string;
  reloadVersion: number;
  disabled: boolean;
  onSelect: (name: string, descendantNames: string[]) => void;
}) {
  return renderCatalogCategoryNav(html`
    <category-tree-sidebar
      scope="tool"
      .items=${props.items}
      .selectedCategory=${props.selectedCategory || "__all__"}
      .keyword=${props.keyword}
      .gatewayHost=${props.gatewayHost ?? ""}
      .token=${props.token ?? ""}
      .reloadVersion=${props.reloadVersion}
      ?disabled=${props.disabled}
      @category-select=${(e: CustomEvent<{ name: string; descendantNames?: string[] }>) => {
        props.onSelect(e.detail.name, e.detail.descendantNames ?? []);
      }}
    ></category-tree-sidebar>
  `);
}

export function renderEmployeeCategoryNav(props: {
  items: import("../controllers/remote-market.ts").EmployeeListItem[];
  selectedCategory: string;
  keyword: string;
  gatewayHost?: string;
  token?: string;
  reloadVersion: number;
  disabled: boolean;
  onSelect: (name: string, descendantNames: string[]) => void;
}) {
  return renderCatalogCategoryNav(html`
    <category-tree-sidebar
      scope="employee"
      .items=${props.items}
      .selectedCategory=${props.selectedCategory || "__all__"}
      .keyword=${props.keyword}
      .gatewayHost=${props.gatewayHost ?? ""}
      .token=${props.token ?? ""}
      .reloadVersion=${props.reloadVersion}
      ?disabled=${props.disabled}
      @category-select=${(e: CustomEvent<{ name: string; descendantNames?: string[] }>) => {
        props.onSelect(e.detail.name, e.detail.descendantNames ?? []);
      }}
    ></category-tree-sidebar>
  `);
}

export function renderModelCategoryNav(props: {
  providers: Record<string, ModelProvider>;
  providerSearchQuery: string;
  selectedCategory: ModelLibraryCategory;
  disabled: boolean;
  embeddedPlazaCount?: number;
  embeddedInstalledCount?: number;
  onCategoryChange: (category: ModelLibraryCategory) => void;
}) {
  const plazaCount = props.embeddedPlazaCount ?? 3;
  const embeddedInstalled = props.embeddedInstalledCount ?? 0;
  const { orderedCategories, counts } = computeModelLibraryCategories(
    props.providers,
    props.providerSearchQuery,
    plazaCount,
    embeddedInstalled,
  );
  const effectiveCategory = props.selectedCategory ?? "__all__";

  const labelFor = (catKey: ModelLibraryCategory) => {
    switch (catKey) {
      case "__all__":
        return "全部";
      case "plaza":
        return "本机模型";
      case "public":
        return "公有模型";
      case "local":
        return "本地模型";
      default:
        return catKey;
    }
  };

  return renderCatalogCategoryNav(html`
    <div class="emp-categories">
      ${orderedCategories.map((catKey) => {
        const label = labelFor(catKey);
        const active = effectiveCategory === catKey;
        const count = counts.get(catKey) ?? 0;
        return html`
          <button
            class="emp-cat ${active ? "active" : ""}"
            type="button"
            ?disabled=${props.disabled}
            @click=${() => props.onCategoryChange(catKey)}
          >
            <span class="emp-cat__name">${label}</span>
            <span class="emp-cat__count">${count}</span>
          </button>
        `;
      })}
    </div>
  `);
}
