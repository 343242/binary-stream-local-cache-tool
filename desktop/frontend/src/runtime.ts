type RuntimeWindow = Window & {
  runtime?: {
    EventsOn?: (eventName: string, callback: (...data: any[]) => void) => () => void;
  };
};

export function hasRuntime() {
  const runtimeWindow = window as RuntimeWindow;
  return Boolean(runtimeWindow.runtime?.EventsOn);
}

export function subscribeToEvent(eventName: string, callback: (...data: any[]) => void): () => void {
  const runtimeWindow = window as RuntimeWindow;
  const unsubscribe = runtimeWindow.runtime?.EventsOn?.(eventName, callback);
  return unsubscribe ?? (() => undefined);
}
