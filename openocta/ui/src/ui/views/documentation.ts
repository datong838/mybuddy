import { html, nothing } from "lit";

/** 默认在线文档（语雀）地址 */
export const ONLINE_DOCUMENTATION_URL =
  "https://databuff.yuque.com/org-wiki-databuff-spr8e6/lqn7on";

export type DocumentationViewProps = {
  url?: string;
  onOpenExternal?: () => void;
};

export function renderDocumentation(props: DocumentationViewProps = {}) {
  const url = props.url ?? ONLINE_DOCUMENTATION_URL;

  return html`
    <section class="documentation-embed">
      ${props.onOpenExternal
        ? html`
            <div class="row" style="justify-content: flex-end; margin-bottom: 12px;">
              <button type="button" class="btn btn--sm" @click=${props.onOpenExternal}>
                在新窗口打开
              </button>
            </div>
          `
        : nothing}
      <div class="documentation-embed__frame-wrap">
        <iframe
          class="documentation-embed__frame"
          src=${url}
          referrerpolicy="no-referrer-when-downgrad"
        ></iframe>
      </div>
    </section>
  `;
}
