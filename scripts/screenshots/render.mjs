import { readdir, readFile, mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
import { chromium } from 'playwright';

const require = createRequire(import.meta.url);
const [framesDir, outDir] = process.argv.slice(2);
const scale = 2;
const px = (size) => `${size * scale}px`;

if (!framesDir || !outDir) {
  console.error('usage: node render.mjs <frames-dir> <out-dir>');
  process.exit(1);
}

const xtermJs = await readFile(require.resolve('@xterm/xterm/lib/xterm.js'), 'utf8');
const canvasAddonJs = await readFile(require.resolve('@xterm/addon-canvas/lib/addon-canvas.js'), 'utf8');
const xtermCss = await readFile(require.resolve('@xterm/xterm/css/xterm.css'), 'utf8');

const page = (frame, cols, rows) => `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700&display=block">
<style>
${xtermCss}
body { margin: 0; padding: ${px(40)}; background: transparent; font-family: 'JetBrains Mono', monospace; }
.window { display: inline-block; overflow: hidden; border-radius: ${px(12)}; border: ${px(1)} solid #605e5c99; background: #1b1a19; box-shadow: 0 ${px(20)} ${px(50)} #0009; }
.titlebar { position: relative; display: flex; align-items: center; gap: ${px(8)}; height: ${px(38)}; padding: 0 ${px(16)}; background: #252423; border-bottom: ${px(1)} solid #605e5c66; }
.dot { width: ${px(12)}; height: ${px(12)}; border-radius: 50%; }
.title { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; font-size: ${px(13)}; color: #a19f9d; }
.screen { padding: ${px(16)} ${px(18)}; }
.xterm-viewport { overflow: hidden !important; }
</style>
</head>
<body>
<div class="window" id="window">
  <div class="titlebar">
    <span class="dot" style="background:#e0475b"></span>
    <span class="dot" style="background:#f2cb1d"></span>
    <span class="dot" style="background:#6ccb5f"></span>
    <span class="title">lazyado — ${cols}×${rows}</span>
  </div>
  <div class="screen"><div id="terminal"></div></div>
</div>
<script>${xtermJs}</script>
<script>${canvasAddonJs}</script>
<script>
const frame = ${JSON.stringify(frame)};
Promise.all([
  document.fonts.load('14px "JetBrains Mono"'),
  document.fonts.load('bold 14px "JetBrains Mono"'),
]).then(() => {
  const term = new Terminal({
    cols: ${cols},
    rows: ${rows},
    fontFamily: '"JetBrains Mono"',
    fontSize: ${14 * scale},
    theme: { background: '#1b1a19', foreground: '#f3f2f1' },
  });
  term.open(document.getElementById('terminal'));
  term.loadAddon(new CanvasAddon.CanvasAddon());
  term.write(frame, () => requestAnimationFrame(() => requestAnimationFrame(() => { window.rendered = true; })));
});
</script>
</body>
</html>`;

const browser = await chromium.launch();
const context = await browser.newContext({ deviceScaleFactor: 1, viewport: { width: 3200, height: 2400 } });
await mkdir(outDir, { recursive: true });

const frames = (await readdir(framesDir)).filter((name) => name.endsWith('.ans')).sort();

for (const name of frames) {
  const raw = await readFile(path.join(framesDir, name), 'utf8');
  const lines = raw.split('\n');
  const cols = Math.max(...lines.map((line) => line.replace(/\x1b\[[0-9;]*m/g, '').length));
  const tab = await context.newPage();
  await tab.setContent(page(lines.join('\r\n'), cols, lines.length));
  await tab.waitForFunction(() => window.rendered);
  const out = path.join(outDir, name.replace(/\.ans$/, '.png'));
  await tab.locator('#window').screenshot({ path: out, omitBackground: true });
  await tab.close();
  console.log(out);
}

await browser.close();
