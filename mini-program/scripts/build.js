/**
 * Native WeChat mini programs are packaged by WeChat DevTools, not by a JS
 * bundler. This "build" step is a structural validation gate so CI / reviewers
 * get a deterministic pass/fail without opening DevTools:
 *
 *   1. app.json is valid JSON and references pages that exist on disk.
 *   2. Every page has the required .js / .json / .wxml files.
 *   3. Every usingComponents path (global + per page/component) resolves.
 *   4. The custom tab bar and the finalized tab icons are present.
 */
const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const MP = path.join(ROOT, 'miniprogram');

const errors = [];
const checked = { pages: 0, components: 0 };

function fail(msg) {
  errors.push(msg);
}

function readJson(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

function exists(p) {
  return fs.existsSync(p);
}

/** Resolve a usingComponents value (absolute "/a/b" or relative) to a file base. */
function resolveComponent(fromDir, ref) {
  if (ref.startsWith('plugin://') || ref.startsWith('weui://')) return null;
  const base = ref.startsWith('/') ? path.join(MP, ref) : path.resolve(fromDir, ref);
  return base;
}

function checkUsingComponents(ownerFile, json) {
  const using = json.usingComponents || {};
  const dir = path.dirname(ownerFile);
  Object.entries(using).forEach(([name, ref]) => {
    const base = resolveComponent(dir, ref);
    if (base === null) return;
    if (!exists(`${base}.js`) || !exists(`${base}.wxml`) || !exists(`${base}.json`)) {
      fail(`${path.relative(ROOT, ownerFile)} → component "${name}" (${ref}) not found`);
    } else {
      checked.components += 1;
    }
  });
}

function checkPage(pagePath) {
  const base = path.join(MP, pagePath);
  ['js', 'json', 'wxml'].forEach((ext) => {
    if (!exists(`${base}.${ext}`)) fail(`page ${pagePath}: missing ${pagePath}.${ext}`);
  });
  if (exists(`${base}.json`)) {
    try {
      checkUsingComponents(`${base}.json`, readJson(`${base}.json`));
    } catch (e) {
      fail(`page ${pagePath}: invalid json (${e.message})`);
    }
  }
  checked.pages += 1;
}

function main() {
  const appFile = path.join(MP, 'app.json');
  if (!exists(appFile)) {
    fail('miniprogram/app.json missing');
    return report();
  }
  let app;
  try {
    app = readJson(appFile);
  } catch (e) {
    fail(`app.json invalid JSON: ${e.message}`);
    return report();
  }

  const pages = [].concat(app.pages || []);
  (app.subPackages || app.subpackages || []).forEach((sp) => {
    (sp.pages || []).forEach((p) => pages.push(`${sp.root}/${p}`.replace(/\/+/g, '/')));
  });
  if (!pages.length) fail('app.json declares no pages');
  pages.forEach(checkPage);

  // Global usingComponents.
  checkUsingComponents(appFile, app);

  // Custom tab bar.
  if (app.tabBar && app.tabBar.custom) {
    const tb = path.join(MP, 'custom-tab-bar', 'index');
    ['js', 'json', 'wxml', 'wxss'].forEach((ext) => {
      if (!exists(`${tb}.${ext}`)) fail(`custom-tab-bar: missing index.${ext}`);
    });
  }

  // Finalized tab icons.
  ['home', 'reservation', 'order', 'me'].forEach((n) => {
    ['png', 'png'].forEach(() => {});
    if (!exists(path.join(MP, 'assets', 'tab', `${n}.png`))) fail(`tab icon assets/tab/${n}.png missing (run npm run gen:icons)`);
    if (!exists(path.join(MP, 'assets', 'tab', `${n}-active.png`))) fail(`tab icon assets/tab/${n}-active.png missing`);
  });

  report();
}

function report() {
  if (errors.length) {
    console.error('BUILD FAILED:');
    errors.forEach((e) => console.error('  - ' + e));
    process.exit(1);
  }
  console.log(`build OK — ${checked.pages} pages, ${checked.components} component references resolved.`);
}

main();
