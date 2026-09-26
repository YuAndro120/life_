// Тема: выбранная, а вечером и ночью (по умолчанию) — «Сумерки».
export const DUSK_START_HOUR = 19;
export const DUSK_END_HOUR = 6;

export function resolveTheme(settings, now) {
  if (!settings.autoDusk) return settings.theme;
  const h = now.getHours();
  return h >= DUSK_START_HOUR || h < DUSK_END_HOUR ? 'dusk' : settings.theme;
}
