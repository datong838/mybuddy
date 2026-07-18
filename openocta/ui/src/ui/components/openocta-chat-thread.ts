import { customElement, property } from "lit/decorators.js";
import { LitElement, html, nothing } from "lit";
import { repeat } from "lit/directives/repeat.js";
import type * as v0_9 from "@a2ui/web_core/v0_9";
import type { GatewayBrowserClient } from "../gateway.ts";
import type { FilePreviewRequest } from "../chat/file-blocks.ts";
import type { MessageGroup } from "../types/chat-types.ts";
import { buildChatItems } from "../chat/chat-items.ts";
import {
  renderA2UIGroup,
  renderMessageGroup,
  renderReadingIndicatorGroup,
  renderStreamingGroup,
} from "../chat/grouped-render.ts";

export type ChatThreadAssistantIdentity = {
  name: string;
  avatar: string | null;
};

/**
 * Isolated message list: parent draft/compose updates do not re-render this tree.
 */
@customElement("openocta-chat-thread")
export class OpenOctaChatThread extends LitElement {
  createRenderRoot() {
    return this;
  }

  @property() sessionKey = "main";
  @property({ type: Boolean }) loading = false;
  @property({ type: Boolean }) canAbort = false;
  @property({ type: Boolean }) conversationOnly = false;
  @property({ type: Boolean }) showReasoning = false;
  @property() assistantName = "Assistant";
  @property() assistantAvatar: string | null = null;

  @property({ attribute: false }) messages: unknown[] = [];
  @property({ attribute: false }) toolMessages: unknown[] = [];
  @property({ attribute: false }) stream: string | null = null;
  @property({ attribute: false }) reasoningStream: string | null = null;
  @property({ attribute: false }) streamStartedAt: number | null = null;
  @property({ attribute: false }) runPhase: "idle" | "thinking" | "tool" | "streaming" = "idle";
  @property({ attribute: false }) a2uiMessages: unknown[] = [];
  @property({ attribute: false }) client: GatewayBrowserClient | null = null;

  @property({ attribute: false }) emptyIntro: unknown = nothing;

  render() {
    const showToolTrace = !this.conversationOnly;
    const assistantIdentity: ChatThreadAssistantIdentity = {
      name: this.assistantName,
      avatar: this.assistantAvatar,
    };

    return html`
      <div
        class="chat-thread"
        role="log"
        aria-live="off"
        @scroll=${this.#onScroll}
        @click=${this.#onClick}
      >
        ${this.loading ? html`<div class="muted">Loading chat…</div>` : nothing}
        ${repeat(
          buildChatItems({
            sessionKey: this.sessionKey,
            messages: this.messages,
            toolMessages: this.toolMessages,
            conversationOnly: this.conversationOnly,
            canAbort: this.canAbort,
            stream: this.stream,
            streamStartedAt: this.streamStartedAt,
            reasoningStream: this.reasoningStream,
            runPhase: this.runPhase,
            a2uiMessages: this.a2uiMessages,
          }),
          (item) => item.key,
          (item) => this.#renderItem(item, showToolTrace, assistantIdentity),
        )}
        ${this.emptyIntro}
      </div>
    `;
  }

  #onScroll = (event: Event) => {
    this.dispatchEvent(
      new CustomEvent("chat-scroll", { detail: { event }, bubbles: true, composed: true }),
    );
  };

  #onClick = (event: Event) => {
    const target = (event.target as HTMLElement | null)?.closest?.(
      "a[data-chat-attachment]",
    ) as HTMLAnchorElement | null;
    if (!target) {
      return;
    }
    event.preventDefault();
    const path = target.getAttribute("data-chat-attachment");
    if (path) {
      this.dispatchEvent(
        new CustomEvent("open-attachment", { detail: { path }, bubbles: true, composed: true }),
      );
    }
  };

  #onA2UIAction = async (action: v0_9.A2uiClientAction) => {
    this.dispatchEvent(
      new CustomEvent("a2ui-action", { detail: { action }, bubbles: true, composed: true }),
    );
  };

  #onOpenSidebar = (content: string) => {
    this.dispatchEvent(
      new CustomEvent("open-sidebar", { detail: { content }, bubbles: true, composed: true }),
    );
  };

  #onFilePreview = (req: FilePreviewRequest) => {
    this.dispatchEvent(
      new CustomEvent("file-preview", { detail: { req }, bubbles: true, composed: true }),
    );
  };

  #renderItem(
    item: ReturnType<typeof buildChatItems>[number],
    showToolTrace: boolean,
    assistantIdentity: ChatThreadAssistantIdentity,
  ) {
    if (item.kind === "reading-indicator") {
      return renderReadingIndicatorGroup(assistantIdentity, item.startedAt, item.phase, item.reasoningText);
    }
    if (item.kind === "stream") {
      return renderStreamingGroup(
        item.text,
        item.startedAt,
        this.#onOpenSidebar,
        assistantIdentity,
        item.reasoningText,
      );
    }
    if (item.kind === "a2ui") {
      return renderA2UIGroup(
        item.messages,
        assistantIdentity,
        this.client,
        this.sessionKey,
        (action) => this.#onA2UIAction(action),
        (req) => this.#onFilePreview(req),
        {
          runPhase: item.runPhase,
          reasoningText: item.reasoningText,
        },
      );
    }
    if (item.kind === "group") {
      const group = item as MessageGroup;
      return renderMessageGroup(
        group,
        {
          onOpenSidebar: this.#onOpenSidebar,
          onFilePreview: (req) => this.#onFilePreview(req),
          showReasoning: this.showReasoning,
          assistantName: assistantIdentity.name,
          assistantAvatar: assistantIdentity.avatar ?? undefined,
          client: this.client,
          sessionKey: this.sessionKey,
          onA2UIAction: (action) => this.#onA2UIAction(action),
          showToolTrace,
        } as never,
      );
    }
    return nothing;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "openocta-chat-thread": OpenOctaChatThread;
  }
}
