import assert from 'node:assert/strict';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { readInventory, verifyInventory } from './windows-concurrency-tests.mjs';

test('additive manifests keep package-qualified requirements', () => {
  const dir = mkdtempSync(join(tmpdir(), 'concurrency-inventory-'));
  try {
    const a = 'reasonix/internal/first', b = 'reasonix/internal/second';
    writeFileSync(join(dir, 'one.json'), JSON.stringify([{ package: a, tests: ['TestShared', 'TestShared/case'] }]));
    writeFileSync(join(dir, 'two.json'), JSON.stringify([{ package: b, tests: ['TestShared'] }]));
    const required = readInventory(dir);
    const events = [
      { Package: a, Action: 'pass', Test: 'TestShared/case' },
      { Package: a, Action: 'pass', Test: 'TestShared' },
      { Package: a, Action: 'pass' },
      { Package: b, Action: 'pass', Test: 'TestShared' },
      { Package: b, Action: 'pass' },
    ];
    const raw = values => values.map(value => JSON.stringify(value)).join('\r\n');
    assert.deepEqual(verifyInventory(required, raw(events)), { requiredTests: 3, packages: 2 });
    for (let i = 0; i < events.length; i++) {
      assert.throws(() => verifyInventory(required, raw(events.filter((_, index) => index !== i))));
      assert.throws(() => verifyInventory(required, raw(events.map((event, index) => index === i ? { ...event, Action: 'skip' } : event))));
    }
    assert.throws(() => verifyInventory(required, raw([...events, { Package: a, Action: 'fail' }])));
    assert.throws(() => verifyInventory(required, raw([...events, events[0]])));
    assert.throws(() => verifyInventory(required, '{'));
    writeFileSync(join(dir, 'two.json'), JSON.stringify([{ package: b, tests: ['TestOrphan/case'] }]));
    assert.throws(() => readInventory(dir));
    writeFileSync(join(dir, 'two.json'), JSON.stringify([{ package: '../other', tests: ['TestShared'] }]));
    assert.throws(() => readInventory(dir));
    writeFileSync(join(dir, 'two.json'), '[]');
    assert.throws(() => readInventory(dir));
    rmSync(join(dir, 'one.json'));
    rmSync(join(dir, 'two.json'));
    assert.throws(() => readInventory(dir));
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
