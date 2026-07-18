/**
 * 桌面端（Wails WebView2 等）中 window.confirm / alert / prompt 可能不显示或阻塞异常。
 * 由 OpenClawApp 注册应用内浮层实现；未注册时回退到浏览器原生 API。
 */
export type NativeDialogInvoker = {
  showConfirm(message: string): Promise<boolean>;
  showAlert(message: string): Promise<void>;
  showPrompt(message: string, defaultValue?: string): Promise<string | null>;
};

let invoker: NativeDialogInvoker | null = null;

export function registerNativeDialogInvoker(next: NativeDialogInvoker) {
  invoker = next;
}

export function unregisterNativeDialogInvoker(next: NativeDialogInvoker) {
  if (invoker === next) {
    invoker = null;
  }
}

export async function nativeConfirm(message: string): Promise<boolean> {
  if (invoker) {
    return invoker.showConfirm(message);
  }
  return window.confirm(message);
}

export async function nativeAlert(message: string): Promise<void> {
  if (invoker) {
    await invoker.showAlert(message);
    return;
  }
  window.alert(message);
}

export async function nativePrompt(message: string, defaultValue = ""): Promise<string | null> {
  if (invoker) {
    return invoker.showPrompt(message, defaultValue);
  }
  return window.prompt(message, defaultValue);
}
