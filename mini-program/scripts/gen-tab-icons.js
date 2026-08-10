/**
 * Rasterize the finalized tab-bar SVGs (design/mini-program/tab-icons/) into
 * PNG assets consumable by the WeChat custom tab bar.
 *
 * We DO NOT invent new icons — we only take the design-frozen SVG line icons
 * and produce two color variants per the tab-icons README:
 *   - normal   : dark-gray thin line  (#8A8A8A)
 *   - selected : pure black           (#111111)
 *
 * Output: miniprogram/assets/tab/<name>.png and <name>-active.png (81x81).
 */
const fs = require('fs');
const path = require('path');
const sharp = require('sharp');

const REPO_ROOT = path.resolve(__dirname, '..', '..');
const SRC_DIR = path.join(REPO_ROOT, 'design', 'mini-program', 'tab-icons');
const OUT_DIR = path.resolve(__dirname, '..', 'miniprogram', 'assets', 'tab');

const ICONS = ['home', 'reservation', 'order', 'me'];
const SIZE = 81; // WeChat recommended tab icon size

function recolor(svg, color) {
  return svg
    .replace(/stroke="#[0-9a-fA-F]{3,6}"/g, `stroke="${color}"`)
    .replace(/fill="#[0-9a-fA-F]{3,6}"/g, `fill="${color}"`);
}

async function render(svg, outFile) {
  await sharp(Buffer.from(svg), { density: 384 })
    .resize(SIZE, SIZE, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
    .png()
    .toFile(outFile);
  console.log('wrote', path.relative(REPO_ROOT, outFile));
}

async function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  for (const name of ICONS) {
    const src = path.join(SRC_DIR, `${name}.svg`);
    if (!fs.existsSync(src)) {
      throw new Error(`missing finalized tab icon: ${src}`);
    }
    const svg = fs.readFileSync(src, 'utf8');
    await render(recolor(svg, '#8A8A8A'), path.join(OUT_DIR, `${name}.png`));
    await render(recolor(svg, '#111111'), path.join(OUT_DIR, `${name}-active.png`));
  }
  console.log('tab icons generated.');
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
