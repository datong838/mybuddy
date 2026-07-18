import { customElement, property, state } from "lit/decorators.js";
import { LitElement, html, nothing } from "lit";
import { localAgentInitial, primaryMentionAlias, type LocalAgentProbeResult } from "../local-agents.ts";

function adjustTextareaHeight(el: HTMLTextAreaElement) {
  el.style.height = "auto";
  el.style.height = `${el.scrollHeight}px`;
}

type MentionMatch = {
  start: number;
  end: number;
  query: string;
};

function findActiveMention(text: string, caret: number): MentionMatch | null {
  const before = text.slice(0, caret);
  const at = before.lastIndexOf("@");
  if (at < 0) {
    return null;
  }
  const between = before.slice(at + 1);
  if (!/^[\w-]*$/.test(between)) {
    return null;
  }
  if (at > 0) {
    const prev = before[at - 1];
    if (prev && !/\s/.test(prev)) {
      return null;
    }
  }
  return { start: at, end: caret, query: between.toLowerCase() };
}

/**
 * Draft input with local state so keystrokes do not re-render the whole chat page.
 */
@customElement("openocta-chat-input")
export class OpenOctaChatInput extends LitElement {
  createRenderRoot() {
    return this;
  }

  @property({ type: Boolean }) disabled = false;
  @property() placeholder = "";
  /** Increment after send/clear to reset the textarea without binding every keystroke to the app. */
  @property({ type: Number }) clearToken = 0;
  /** Optional external prefill (quick prompts). */
  @property() prefill = "";
  /** Installed local agents for @ mention autocomplete. */
  @property({ attribute: false }) mentionOptions: LocalAgentProbeResult[] = [];
  /** Increment to append insertSnippet to the draft. */
  @property({ type: Number }) insertToken = 0;
  @property() insertSnippet = "";

  @state() private localDraft = "";
  @state() private mentionOpen = false;
  @state() private mentionQuery = "";
  @state() private mentionStart = 0;
  @state() private mentionActiveIndex = 0;

  #lastClearToken = 0;
  #lastPrefill = "";
  #lastInsertToken = 0;

  protected willUpdate(changed: Map<PropertyKey, unknown>): void {
    if (changed.has("clearToken") && this.clearToken !== this.#lastClearToken) {
      this.#lastClearToken = this.clearToken;
      this.localDraft = "";
      this.#lastPrefill = "";
      this.closeMention();
      this.dispatchEvent(
        new CustomEvent("compose-draft-change", {
          detail: { hasText: false },
          bubbles: true,
          composed: true,
        }),
      );
    }
    if (
      changed.has("prefill") &&
      this.prefill &&
      this.prefill !== this.#lastPrefill &&
      this.prefill !== this.localDraft
    ) {
      this.#lastPrefill = this.prefill;
      this.localDraft = this.prefill;
    }
    if (changed.has("insertToken") && this.insertToken !== this.#lastInsertToken) {
      this.#lastInsertToken = this.insertToken;
      const snippet = this.insertSnippet;
      if (snippet) {
        this.localDraft = this.localDraft ? `${this.localDraft}${snippet}` : snippet;
        this.dispatchDraftChange();
      }
    }
  }

  protected updated(changed: Map<PropertyKey, unknown>): void {
    if (changed.has("localDraft") || changed.has("clearToken") || changed.has("prefill")) {
      const textarea = this.renderRoot.querySelector("textarea");
      if (textarea instanceof HTMLTextAreaElement) {
        adjustTextareaHeight(textarea);
      }
    }
  }

  focusInput() {
    const textarea = this.renderRoot.querySelector("textarea");
    if (textarea instanceof HTMLTextAreaElement) {
      textarea.focus();
    }
  }

  getDraft(): string {
    return this.localDraft;
  }

  private filteredMentions(): LocalAgentProbeResult[] {
    const q = this.mentionQuery.trim().toLowerCase();
    const installed = this.mentionOptions.filter((a) => a.installed);
    if (!q) {
      return installed;
    }
    return installed.filter((a) => {
      if (a.id.includes(q) || a.label.toLowerCase().includes(q)) {
        return true;
      }
      return a.aliases.some((alias) => alias.toLowerCase().includes(q));
    });
  }

  private closeMention() {
    this.mentionOpen = false;
    this.mentionQuery = "";
    this.mentionActiveIndex = 0;
  }

  private syncMentionFromCaret(textarea: HTMLTextAreaElement) {
    const match = findActiveMention(this.localDraft, textarea.selectionStart ?? this.localDraft.length);
    if (!match || this.mentionOptions.length === 0) {
      this.closeMention();
      return;
    }
    this.mentionOpen = true;
    this.mentionQuery = match.query;
    this.mentionStart = match.start;
    this.mentionActiveIndex = 0;
  }

  private applyMention(agent: LocalAgentProbeResult) {
    const alias = primaryMentionAlias(agent);
    const before = this.localDraft.slice(0, this.mentionStart);
    const afterStart = this.mentionStart + 1 + this.mentionQuery.length;
    const after = this.localDraft.slice(afterStart);
    this.localDraft = `${before}@${alias} ${after}`;
    this.closeMention();
    this.dispatchDraftChange();
    requestAnimationFrame(() => {
      const textarea = this.renderRoot.querySelector("textarea");
      if (textarea instanceof HTMLTextAreaElement) {
        const pos = before.length + alias.length + 2;
        textarea.focus();
        textarea.setSelectionRange(pos, pos);
        adjustTextareaHeight(textarea);
      }
    });
  }

  private dispatchDraftChange() {
    this.dispatchEvent(
      new CustomEvent("compose-draft-change", {
        detail: { hasText: this.localDraft.trim().length > 0 },
        bubbles: true,
        composed: true,
      }),
    );
  }

  render() {
    const options = this.filteredMentions();
    return html`
      <span class="textarea chat-compose__textarea">
        <textarea
          rows="1"
          .value=${this.localDraft}
          ?disabled=${this.disabled}
          placeholder=${this.placeholder}
          @keydown=${this.#onKeydown}
          @input=${this.#onInput}
          @paste=${this.#onPaste}
          @click=${this.#onCaretSync}
          @keyup=${this.#onCaretSync}
        ></textarea>
        ${
          this.mentionOpen && options.length > 0
            ? html`
                <div class="chat-mention-popover" role="listbox">
                  ${options.map(
                    (agent, index) => html`
                      <button
                        type="button"
                        class="chat-mention-popover__item ${index === this.mentionActiveIndex
                          ? "chat-mention-popover__item--active"
                          : ""}"
                        role="option"
                        @mousedown=${(e: Event) => {
                          e.preventDefault();
                          this.applyMention(agent);
                        }}
                      >
                        <span class="local-agent-chip__avatar local-agent-chip__avatar--sm" aria-hidden="true"
                          >${localAgentInitial(agent.id)}</span
                        >
                        <span>@${primaryMentionAlias(agent)}</span>
                      </button>
                    `,
                  )}
                </div>
              `
            : nothing
        }
      </span>
    `;
  }

  #onInput = (event: Event) => {
    const target = event.target as HTMLTextAreaElement;
    this.localDraft = target.value;
    adjustTextareaHeight(target);
    this.syncMentionFromCaret(target);
    this.dispatchDraftChange();
  };

  #onCaretSync = (event: Event) => {
    const target = event.target as HTMLTextAreaElement;
    this.syncMentionFromCaret(target);
  };

  #onKeydown = (event: KeyboardEvent) => {
    if (this.mentionOpen) {
      const options = this.filteredMentions();
      if (options.length > 0) {
        if (event.key === "ArrowDown") {
          event.preventDefault();
          this.mentionActiveIndex = (this.mentionActiveIndex + 1) % options.length;
          return;
        }
        if (event.key === "ArrowUp") {
          event.preventDefault();
          this.mentionActiveIndex = (this.mentionActiveIndex - 1 + options.length) % options.length;
          return;
        }
        if (event.key === "Enter" || event.key === "Tab") {
          event.preventDefault();
          this.applyMention(options[this.mentionActiveIndex]!);
          return;
        }
      }
      if (event.key === "Escape") {
        event.preventDefault();
        this.closeMention();
        return;
      }
    }

    if (event.key !== "Enter") {
      return;
    }
    if (event.isComposing || event.keyCode === 229) {
      return;
    }
    if (event.shiftKey) {
      return;
    }
    if (this.disabled || this.mentionOpen) {
      return;
    }
    event.preventDefault();
    this.#emitSend();
  };

  requestSend() {
    this.#emitSend();
  }

  #emitSend() {
    this.dispatchEvent(
      new CustomEvent("compose-send", {
        detail: { message: this.localDraft },
        bubbles: true,
        composed: true,
      }),
    );
  }

  #onPaste = (event: ClipboardEvent) => {
    this.dispatchEvent(
      new CustomEvent("compose-paste", {
        detail: { event },
        bubbles: true,
        composed: true,
      }),
    );
  };
}

declare global {
  interface HTMLElementTagNameMap {
    "openocta-chat-input": OpenOctaChatInput;
  }
}
