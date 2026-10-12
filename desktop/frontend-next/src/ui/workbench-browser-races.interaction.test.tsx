// @vitest-environment jsdom
import { act, cleanup, fireEvent, render } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import './testkit';
import { WorkbenchPanel } from './WorkbenchPanel';
import { MockPort } from '../port/mock';
import { host } from '../port/host';
import { openPage } from './browser_open';
const page = { id: 'b1', target: 't1', url: 'https://example.com', title: 'Existing page', active: true };
const props = { scheme: 'dark' as const, shown: false, changes: [], onSurfaces: vi.fn(), onCloseManual: vi.fn(), onExternal: vi.fn() };
const filesShown = () => document.querySelector('.workbench-body')!.hasAttribute('data-files');
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
it('keeps Files visible when late browser tab list announces older followed link', async () => {
  const port = new MockPort();
  let resolve!: (v: typeof page) => void;
  vi.spyOn(port, 'browserOpen').mockReturnValue(new Promise(done => { resolve = done; }));
  const pending = openPage(port, page.url);
  const { rerender } = render(<WorkbenchPanel {...props} port={port} tabs={[]} manual />);
  rerender(<WorkbenchPanel {...props} port={port} tabs={[]} manual revealChanges={1} />);
  expect(filesShown()).toBe(true);
  await act(async () => { resolve(page); await pending; });
  expect(filesShown()).toBe(true);
  rerender(<WorkbenchPanel {...props} port={port} tabs={[page]} manual revealChanges={1} />);
  expect(filesShown()).toBe(true);
});
it.each(['workbench.launch-browser', 'workbench.new-browser'])('keeps Files visible when earlier %s finishes', async (action) => {
  vi.spyOn(host(), 'drawsBrowserViews').mockReturnValue(true);
  const port = new MockPort();
  let resolve!: (v: typeof page) => void;
  const pending = new Promise<typeof page>(done => { resolve = done; });
  vi.spyOn(port, 'browserOpen').mockReturnValue(pending);
  const { rerender } = render(<WorkbenchPanel {...props} port={port} tabs={[]} manual />);
  fireEvent.click(document.querySelector(`[data-action="${action}"]`)!);
  expect(port.browserOpen).toHaveBeenCalledOnce();
  rerender(<WorkbenchPanel {...props} port={port} tabs={[]} manual revealChanges={1} />);
  expect(filesShown()).toBe(true);
  await act(async () => { resolve(page); await pending; });
  expect(filesShown()).toBe(true);
});

import { useState } from 'react';
import { waitFor } from '@testing-library/react';
import { Pane } from './Pane';
import type { WireEvent } from '../port/wire';
const paneProps = { title: 'w', active: true, visible: true, sideHost: null, side: false, onFocus() {}, onReport() {}, onSessionChanged() {}, pulse: 0, findPulse: 0, onSettings() {}, needsProject: false, onOpenProject() {}, onKeepHere() {}, theme: 'dark' as const, dockW: 560, dockMax: 880, onDockW() {} };
it.each(['before', 'after'])('keeps Files visible when Pane receives browser_tabs_changed %s the response', async (order) => {
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null);
  const port = new MockPort();
  let emit!: (event: WireEvent) => void;
  const subscribe = port.subscribe.bind(port);
  vi.spyOn(port, 'subscribe').mockImplementation((onEvent, onGap, bootstrap) => { emit = onEvent; return subscribe(onEvent, onGap, bootstrap); });
  const readTabs = vi.spyOn(port, 'browserTabs').mockResolvedValue([]);
  let resolve!: (v: typeof page) => void;
  vi.spyOn(port, 'browserOpen').mockReturnValue(new Promise(done => { resolve = done; }));
  const pending = openPage(port, page.url);
  function Harness({filesPulse}: {filesPulse: number}) {
    const [manual, setManual] = useState(true);
    return <Pane {...paneProps} rt={{id: 'event-pane', base: '', root: '/w', name: 'w'}} port={port} filesPulse={filesPulse} manualBrowser={manual} onManualBrowser={setManual} />;
  }
  const { rerender } = render(<Harness filesPulse={0} />);
  await waitFor(() => expect(readTabs).toHaveBeenCalledTimes(1));
  await act(async () => {});
  rerender(<Harness filesPulse={1} />);
  expect(filesShown()).toBe(true);
  if (order === 'after') await act(async () => { resolve(page); await pending; });
  expect(filesShown()).toBe(true);
  readTabs.mockResolvedValue([page]);
  await act(async () => { emit({ kind: 'browser_tabs_changed' } as WireEvent); });
  expect(readTabs).toHaveBeenCalledTimes(2);
  if (order === 'before') await act(async () => { resolve(page); await pending; });
  expect(filesShown()).toBe(true);
});

it.each([false, true])('ignores superseded user opens while following a new agent page (same batch=%s)', async (together) => {
  const port = new MockPort();
  const resolve: ((tab: typeof page) => void)[] = [];
  vi.spyOn(port, 'browserOpen').mockImplementation(() => new Promise((done) => { resolve.push(done); }));
  const first = openPage(port, page.url);
  const second = openPage(port, 'https://example.org');
  const later = { ...page, id: 'b2', target: 't2', url: 'https://example.org' };
  const agent = { ...page, id: 'b3', target: 't3', url: 'https://example.net', title: 'New agent page' };
  const { rerender } = render(<WorkbenchPanel {...props} port={port} tabs={[]} manual />);
  rerender(<WorkbenchPanel {...props} port={port} tabs={[]} manual revealChanges={1} />);
  await act(async () => { resolve[0](page); resolve[1](later); await Promise.all([first, second]); });
  if (!together) {
    rerender(<WorkbenchPanel {...props} port={port} tabs={[page, later]} manual revealChanges={1} />);
    await act(async () => {});
    expect(filesShown()).toBe(true);
  }
  rerender(<WorkbenchPanel {...props} port={port} tabs={[agent, page, later]} manual revealChanges={1} />);
  await waitFor(() => expect(filesShown()).toBe(false));
  expect(document.querySelector('[data-target="browser:t3"]')?.closest('[role="tab"]')?.getAttribute('aria-selected')).toBe('true');
});
