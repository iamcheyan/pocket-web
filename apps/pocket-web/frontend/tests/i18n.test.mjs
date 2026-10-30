import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const source = await readFile(new URL('../src/i18n.ts', import.meta.url), 'utf8');
const localeObjects = [...source.matchAll(/^\s{2}(en|zh|ja):\s*\{([^\n]*)\},?$/gm)];

test('English, Simplified Chinese, and Japanese define the same message keys', () => {
  assert.equal(localeObjects.length, 3, 'expected exactly en, zh, and ja message dictionaries');
  const dictionaries = Object.fromEntries(localeObjects.map(([, locale, body]) => [
    locale,
    new Set([...body.matchAll(/(?:^|,\s*)([A-Za-z][A-Za-z0-9_]*):/g)].map(([, key]) => key)),
  ]));
  const expected = dictionaries.en;
  assert.ok(expected.size > 0, 'English dictionary should contain message keys');
  for (const locale of ['zh', 'ja']) {
    assert.deepEqual([...dictionaries[locale]].sort(), [...expected].sort(), `${locale} key parity`);
  }
});
