// Тема: выбранная, а вечером и ночью (по умолчанию) — «Сумерки».
export const DUSK_START_HOUR = 19;
export const DUSK_END_HOUR = 6;

export function resolveTheme(settings, now) {
  if (!settings.autoDusk) return settings.theme;
  const h = now.getHours();
  return h >= DUSK_START_HOUR || h < DUSK_END_HOUR ? 'dusk' : settings.theme;
}

/** Цвета тем для мини-макетов в выборе темы (совпадают с CSS-переменными и с iOS-приложением). */
export const THEME_TOKENS = {
  paper: { bg: '#f2f1ec', ink: '#22211e', muted: '#66655f', line: '#d8d6ce', accent: '#2432d0', plate: '#e4e7f6', card: '#fbfbfe', dark: false },
  sage: { bg: '#edf0ea', ink: '#1e2621', muted: '#5b665e', line: '#d2dad1', accent: '#2f6b4f', plate: '#dae7dd', card: '#fafcf9', dark: false },
  dusk: { bg: '#1c1d22', ink: '#eceae4', muted: '#a3a19b', line: '#34363e', accent: '#a9b3ff', plate: '#262937', card: '#30344a', dark: true },
};
