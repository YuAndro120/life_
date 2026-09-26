// Диагностика раскладки на устройстве: пять быстрых касаний у верхнего края экрана показывают размеры окна и безопасных зон.
export function installDebugPanel() {
  let taps = [];
  let panel = null;
  let timer = null;
  const probe = document.createElement('div');
  probe.style.cssText = 'position:fixed;visibility:hidden;pointer-events:none;padding:env(safe-area-inset-top) env(safe-area-inset-right) env(safe-area-inset-bottom) env(safe-area-inset-left)';
  document.body.append(probe);
  const rect = (sel) => { const el = document.querySelector(sel); if (!el) return '—'; const r = el.getBoundingClientRect(); return `${Math.round(r.top)}…${Math.round(r.bottom)} (h${Math.round(r.height)})`; };
  const lines = () => {
    const cs = getComputedStyle(probe);
    const vv = window.visualViewport;
    return [
      `inner ${innerWidth}×${innerHeight}  screen ${screen.width}×${screen.height}`,
      `visual ${vv ? `${Math.round(vv.width)}×${Math.round(vv.height)} @${Math.round(vv.offsetTop)}` : '—'}  dpr ${devicePixelRatio}`,
      `safe top ${cs.paddingTop} bottom ${cs.paddingBottom}`,
      `standalone ${navigator.standalone === true}`,
      `#app ${rect('#app')}`, `.scroll ${rect('#scroll')}`, `.bar ${rect('#app .bar')}`, `#statusbar ${rect('#statusbar')}`,
      `cap ${getComputedStyle(document.getElementById('statusbar')).backgroundColor}`,
    ].join('\n');
  };
  const toggle = () => {
    if (panel) { panel.remove(); panel = null; clearInterval(timer); return; }
    panel = document.createElement('pre');
    panel.style.cssText = 'position:fixed;left:8px;right:8px;top:70px;z-index:999;margin:0;padding:10px;border-radius:10px;background:#000c;color:#7f7;font:11px/1.35 monospace;white-space:pre-wrap;pointer-events:none';
    document.body.append(panel);
    const draw = () => { panel.textContent = lines(); };
    draw();
    timer = setInterval(draw, 700);
  };
  document.addEventListener('touchstart', (e) => {
    const y = e.touches[0]?.clientY ?? 999;
    if (y > 90) return;
    const now = Date.now();
    taps = [...taps.filter((t) => now - t < 2500), now];
    if (taps.length >= 5) { taps = []; toggle(); }
  }, { passive: true });
}
