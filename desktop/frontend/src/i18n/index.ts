import { defaultLocale, messages, type LocaleKey, type MessageSet } from "./messages";

export type { LocaleKey, MessageSet };
export { defaultLocale };

export function getMessages(locale: LocaleKey): MessageSet {
  return messages[locale] ?? messages[defaultLocale];
}

export function nextLocale(locale: LocaleKey): LocaleKey {
  return locale === "zh-CN" ? "en-US" : "zh-CN";
}

export function formatMessage(template: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce((text, [key, value]) => text.split(`{${key}}`).join(String(value)), template);
}

function lookup<K extends string>(table: Record<string, K>, raw: string) {
  return table[raw] ?? raw;
}

export function localizeWorkspaceMode(locale: LocaleKey, raw: string) {
  return lookup(getMessages(locale).enums.workspaceMode, raw);
}

export function localizeLockMode(locale: LocaleKey, raw: string) {
  return lookup(getMessages(locale).enums.lockMode, raw);
}

export function localizeHealth(locale: LocaleKey, raw: string) {
  return lookup(getMessages(locale).enums.health, raw);
}

export function localizeConfigSection(locale: LocaleKey, raw: string) {
  return lookup(getMessages(locale).enums.configSection, raw);
}

export function localizeConfigField(locale: LocaleKey, raw: string) {
  return lookup(getMessages(locale).enums.configField, raw);
}

export function localizeConfigNote(locale: LocaleKey, raw: string) {
  return lookup(getMessages(locale).enums.configNote, raw);
}
