// Renders HTML files to PDF with headless Chromium, the reference renderer for
// the accuracy gate. Usage: node accuracy/chromium.cjs in.html out.pdf [in2.html out2.pdf ...]
const { chromium } = require('playwright');
const { pathToFileURL } = require('node:url');
const path = require('node:path');

(async () => {
  const args = process.argv.slice(2);
  if (args.length === 0 || args.length % 2 !== 0) {
    console.error('usage: node chromium.cjs in.html out.pdf [...]');
    process.exit(2);
  }
  const opts = process.env.CHROMIUM_PATH ? { executablePath: process.env.CHROMIUM_PATH } : {};
  const browser = await chromium.launch(opts);
  const page = await browser.newPage();
  for (let i = 0; i < args.length; i += 2) {
    await page.goto(pathToFileURL(path.resolve(args[i])).href, { waitUntil: 'load' });
    await page.emulateMedia({ media: 'print' });
    // Same defaults as proteus: A4, 1cm margins, CSS @page wins, backgrounds printed.
    await page.pdf({ path: args[i + 1], preferCSSPageSize: true, printBackground: true, format: 'A4',
      margin: { top: '1cm', right: '1cm', bottom: '1cm', left: '1cm' } });
  }
  await browser.close();
})().catch((e) => { console.error(e); process.exit(1); });
