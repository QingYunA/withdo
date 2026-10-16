import assert from 'node:assert/strict';
import { Buffer } from 'node:buffer';
import { test } from 'node:test';
import { DatabaseSync } from 'node:sqlite';
import { randomUUID } from 'node:crypto';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createAppStore } from '../../src/data/store.ts';

class SQLiteAdapter {
  constructor(path) { this.raw = new DatabaseSync(path); }
  async execAsync(sql) { this.check(sql); this.raw.exec(sql); }
  async runAsync(sql, ...args) { this.check(sql); return this.raw.prepare(sql).run(...args); }
  async getAllAsync(sql, ...args) { return this.raw.prepare(sql).all(...args); }
  check(sql) {
    if (this.fail?.(sql)) { this.fail = undefined; throw new Error('injected SQLite failure'); }
  }
  async withExclusiveTransactionAsync(work) {
    this.raw.exec('BEGIN IMMEDIATE');
    try { await work(this); this.raw.exec('COMMIT'); }
    catch (error) { this.raw.exec('ROLLBACK'); throw error; }
  }
}
class Server {
  epoch = randomUUID();
  cursor = 0;
  records = new Map();
  changes = [];
  receipts = new Map();
  requests = [];
  loseAck = false;
  token = 'private-development-token';
  write(type, id, data) {
    const key = `${type}:${id}`;
    const record = { type, id, version: (this.records.get(key)?.version ?? 0) + 1, deleted: data === null, data };
    this.records.set(key, structuredClone(record));
    this.changes.push({ seq: ++this.cursor, record: structuredClone(record) });
    return record;
  }
  async fetch(url, options) {
    assert.equal(options.headers.Authorization, `Bearer ${this.token}`);
    const parsed = new URL(url);
    if (parsed.pathname.endsWith('/snapshot')) return Response.json({ epoch: this.epoch, cursor: this.cursor, records: [...this.records.values()] });
    if (parsed.pathname.endsWith('/pull')) {
      const cursor = Number(parsed.searchParams.get('cursor'));
      if (parsed.searchParams.get('epoch') !== this.epoch || cursor > this.cursor) return Response.json({ code: 'resnapshot' }, { status: 409 });
      return Response.json({ epoch: this.epoch, cursor: this.cursor, changes: this.changes.filter(change => change.seq > cursor), hasMore: false });
    }
    assert.equal(parsed.pathname, '/v1/sync/push');
    const body = JSON.parse(options.body);
    this.requests.push(structuredClone(body));
    const results = body.mutations.map(mutation => {
      const received = this.receipts.get(mutation.id);
      if (received) {
        assert.deepEqual(received.mutation, mutation, 'a retried mutation ID must retain its exact payload');
        return received.result;
      }
      const record = this.records.get(`${mutation.type}:${mutation.entityId}`) ?? null;
      if ((record?.version ?? 0) !== mutation.baseVersion) return { mutationId: mutation.id, status: 'conflict', record };
      if (!mutation.data && mutation.type === 'task' && [...this.records.values()].some(item => item.type === 'task' && !item.deleted && item.data.parentId === mutation.entityId)) {
        return { mutationId: mutation.id, status: 'rejected', record, error: 'Task still has children' };
      }
      if (mutation.data?.parentId && this.records.get(`task:${mutation.data.parentId}`)?.deleted !== false) {
        return { mutationId: mutation.id, status: 'rejected', record, error: 'Missing parent task' };
      }
      if (mutation.data?.projectId && this.records.get(`project:${mutation.data.projectId}`)?.deleted !== false) {
        return { mutationId: mutation.id, status: 'rejected', record, error: 'Missing project' };
      }
      const result = { mutationId: mutation.id, status: 'accepted', record: this.write(mutation.type, mutation.entityId, mutation.data) };
      this.receipts.set(mutation.id, { mutation: structuredClone(mutation), result });
      return result;
    });
    if (this.loseAck) { this.loseAck = false; throw new TypeError('Network disconnected after commit'); }
    return Response.json({ results });
  }
}
function fixture(t, server = new Server(), extra = {}) {
  const directory = mkdtempSync(join(tmpdir(), 'adhd-mobile-'));
  const path = join(directory, 'tasks.db');
  const tokens = new Map();
  const handles = [];
  let adapter;
  function open() {
    const store = createAppStore({
      openDatabase: async () => { adapter = new SQLiteAdapter(path); handles.push(adapter); return adapter; },
      uuid: randomUUID,
      fetch: server.fetch.bind(server),
      getToken: async url => tokens.get(url) ?? null,
      setToken: async (url, value) => { tokens.set(url, value); },
      ...extra,
    });
    return store;
  }
  const store = open();
  t.after(() => { handles.forEach(handle => { try { handle.raw.close(); } catch {} }); rmSync(directory, { recursive: true, force: true }); });
  return { store, server, open, tokens, get db() { return adapter; }, async connect(target = store) { await target.updateSettings({ serverUrl: 'http://sync.test' }, server.token); } };
}

// These tests run the same store SQL against real SQLite, including actual commit/rollback and file reopen.
test('first run is empty; offline project/task/subtask edits and settings survive a database reopen', async t => {
  const f = fixture(t);
  await f.store.initialize();
  assert.equal(f.store.getSnapshot().tasks.length, 0);
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.equal(f.db.raw.prepare('PRAGMA foreign_keys').get().foreign_keys, 1);
  assert.equal(f.db.raw.prepare('PRAGMA journal_mode').get().journal_mode, 'wal');
  const project = await f.store.createProject('个人项目');
  const task = await f.store.createTask({ title: '离线任务', projectId: project.id });
  const child = await f.store.createTask({ title: '子任务', parentId: task.id });
  await f.store.updateTask(task.id, { status: 'done', notes: '离线笔记' });
  assert.ok(f.store.getSnapshot().tasks.find(item => item.id === task.id).completedAt);
  assert.equal(f.store.getSnapshot().tasks.find(item => item.id === child.id).status, 'done', 'parent completion closes open descendants like desktop');
  await f.store.updateProject(project.id, { archivedAt: new Date().toISOString() });
  await f.store.updateSettings({ language: 'zh', appearance: 'dark' });
  const before = f.store.getSnapshot();
  f.db.raw.close();
  const reopened = f.open();
  await reopened.initialize();
  assert.deepEqual(reopened.getSnapshot(), before);
  await reopened.updateTask(task.id, { status: 'todo' });
  assert.equal(reopened.getSnapshot().tasks.find(item => item.id === task.id).completedAt, null);
});

test('a failed local transaction rolls back both visible data and the outbox', async t => {
  const f = fixture(t);
  const task = await f.store.createTask({ title: 'original' });
  const count = f.store.getSnapshot().pendingCount;
  f.db.fail = sql => sql.startsWith('INSERT INTO tasks');
  await assert.rejects(f.store.updateTask(task.id, { title: 'must roll back' }), /injected/);
  assert.equal(f.store.getSnapshot().tasks[0].title, 'original');
  assert.equal(f.db.raw.prepare('SELECT COUNT(*) AS n FROM outbox').get().n, count);
  const reopened = f.open();
  await reopened.initialize();
  assert.equal(reopened.getSnapshot().tasks[0].title, 'original');
});

test('lost acknowledgement survives reopen, retries the same payload, and rebases the next local edit', async t => {
  const f = fixture(t);
  await f.connect();
  const task = await f.store.createTask({ title: 'initial' });
  f.server.loseAck = true;
  await f.store.sync();
  assert.equal(f.store.getSnapshot().syncState, 'offline');
  await f.store.updateTask(task.id, { title: 'newer local edit' });
  const original = f.server.requests[0].mutations[0];
  f.db.raw.close();
  const reopened = f.open();
  await reopened.initialize();
  await reopened.sync();
  assert.deepEqual(f.server.requests[1].mutations[0], original);
  assert.equal(f.server.records.get(`task:${task.id}`).data.title, 'newer local edit');
  assert.equal(f.server.records.get(`task:${task.id}`).version, 2);
  assert.equal(reopened.getSnapshot().pendingCount, 0);
});

test('inbound changes preserve pending fields, expose conflict, and do not block unrelated tasks', async t => {
  const f = fixture(t);
  await f.connect();
  const task = await f.store.createTask({ title: 'shared', notes: 'old notes' });
  await f.store.sync();
  await f.store.updateTask(task.id, { title: 'local title' });
  const record = f.server.records.get(`task:${task.id}`);
  f.server.write('task', task.id, { ...record.data, notes: 'remote notes', updatedAt: new Date().toISOString() });
  const unrelated = await f.store.createTask({ title: 'unrelated' });
  await f.store.sync();
  const local = f.store.getSnapshot().tasks.find(item => item.id === task.id);
  assert.equal(local.title, 'local title');
  assert.equal(local.notes, 'remote notes');
  assert.equal(f.store.getSnapshot().conflicts.length, 1);
  assert.ok(f.server.records.has(`task:${unrelated.id}`));
  const conflictId = f.store.getSnapshot().conflicts[0].mutation.id;
  await f.store.resolveConflict(conflictId, 'local');
  await f.store.sync();
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.equal(f.server.records.get(`task:${task.id}`).data.title, 'local title');
  assert.equal(f.server.records.get(`task:${task.id}`).data.notes, 'remote notes');
});

test('remote conflict resolution keeps subsequent local intent and retries it against the latest version', async t => {
  const f = fixture(t);
  await f.connect();
  const task = await f.store.createTask({ title: 'shared' });
  await f.store.sync();
  await f.store.updateTask(task.id, { title: 'discard this conflicting edit' });
  await f.store.updateTask(task.id, { notes: 'keep this subsequent edit' });
  const record = f.server.records.get(`task:${task.id}`);
  f.server.write('task', task.id, { ...record.data, title: 'remote title' });
  await f.store.sync();
  const conflict = f.store.getSnapshot().conflicts[0];
  await f.store.resolveConflict(conflict.mutation.id, 'remote');
  await f.store.sync();
  assert.equal(f.store.getSnapshot().conflicts.length, 0);
  assert.equal(f.server.records.get(`task:${task.id}`).data.title, 'remote title');
  assert.equal(f.server.records.get(`task:${task.id}`).data.notes, 'keep this subsequent edit');
});

test('epoch reset resnapshots while preserving unsent offline tasks', async t => {
  const f = fixture(t);
  await f.connect();
  await f.store.sync();
  const task = await f.store.createTask({ title: 'offline across reset' });
  f.server.epoch = randomUUID();
  await f.store.sync();
  assert.equal(f.store.getSnapshot().tasks[0].id, task.id);
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.equal(f.db.raw.prepare("SELECT value FROM metadata WHERE key='epoch'").get().value, f.server.epoch);
});

test('deleting a parent detaches children, keeps them durable, and syncs around older child mutations', async t => {
  const f = fixture(t);
  await f.connect();
  const parent = await f.store.createTask({ title: 'parent' });
  const child = await f.store.createTask({ title: 'child', parentId: parent.id });
  await f.store.updateTask(child.id, { notes: 'pending before detach' });
  await f.store.deleteTask(parent.id);
  assert.equal(f.store.getSnapshot().tasks.length, 1);
  assert.equal(f.store.getSnapshot().tasks[0].parentId, null);
  await f.store.sync();
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.equal(f.server.records.get(`task:${parent.id}`).deleted, true);
  assert.equal(f.server.records.get(`task:${child.id}`).data.parentId, null);
  assert.equal(f.server.records.get(`task:${child.id}`).data.notes, 'pending before detach');
  const reopened = f.open();
  await reopened.initialize();
  assert.equal(reopened.getSnapshot().tasks[0].id, child.id);
});

test('task validation rejects cycles and missing references without creating pending edits', async t => {
  const f = fixture(t);
  const parent = await f.store.createTask({ title: 'parent' });
  const child = await f.store.createTask({ title: 'child', parentId: parent.id });
  const count = f.store.getSnapshot().pendingCount;
  await assert.rejects(f.store.updateTask(parent.id, { parentId: child.id }), /ancestor/);
  await assert.rejects(f.store.createTask({ title: 'orphan', projectId: randomUUID() }), /Project/);
  await assert.rejects(f.store.createTask({ title: 'bad date', remindAt: 'not-a-date' }), /date/);
  assert.equal(f.store.getSnapshot().pendingCount, count);
});

test('server switches are guarded with pending/conflicted changes; token never enters SQLite', async t => {
  const f = fixture(t);
  await f.connect();
  await f.store.createTask({ title: 'keep me' });
  await assert.rejects(f.store.updateSettings({ serverUrl: 'https://other.test' }, 'other-secret'), /pending/);
  assert.equal(f.store.getSnapshot().settings.serverUrl, 'http://sync.test');
  assert.equal(f.store.getSnapshot().pendingCount, 1);
  assert.equal(f.tokens.get('http://sync.test'), f.server.token);
  const allRows = f.db.raw.prepare('SELECT * FROM metadata').all();
  assert.equal(JSON.stringify(allRows).includes(f.server.token), false);
  await f.store.sync();
  await assert.rejects(f.store.updateSettings({ serverUrl: 'https://other.test' }, 'other-secret'), /migration/);
  assert.equal(f.store.getSnapshot().settings.serverUrl, 'http://sync.test');
  assert.equal(f.store.getSnapshot().tasks.length, 1);
  assert.equal(f.tokens.has('https://other.test'), false);
});


test('parent completion closes nested open subtasks and appends reports; reopening keeps children closed', async t => {
  const f = fixture(t);
  await f.connect();
  const project = await f.store.createProject('project');
  const parent = await f.store.createTask({ title: 'parent', projectId: project.id });
  const child = await f.store.createTask({ title: 'child', parentId: parent.id });
  const nested = await f.store.createTask({ title: 'nested', parentId: child.id });
  const cancelled = await f.store.createTask({ title: 'cancelled', parentId: parent.id, status: 'cancelled' });
  assert.equal(child.projectId, project.id, 'a new subtask inherits its parent project');
  await f.store.updateTask(parent.id, { status: 'cancelled' });
  for (const id of [child.id, nested.id]) {
    const task = f.store.getSnapshot().tasks.find(item => item.id === id);
    assert.equal(task.status, 'done');
    assert.ok(task.completedAt);
    assert.equal(f.store.getSnapshot().reports.filter(report => report.taskId === id).length, 1);
  }
  assert.equal(f.store.getSnapshot().tasks.find(item => item.id === cancelled.id).status, 'cancelled');
  assert.equal(f.store.getSnapshot().reports.filter(report => report.taskId === cancelled.id).length, 0);
  await f.store.updateTask(parent.id, { status: 'doing' });
  assert.equal(f.store.getSnapshot().tasks.find(item => item.id === child.id).status, 'done');
  await f.store.sync();
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.equal(f.server.records.get(`task:${nested.id}`).data.status, 'done');
  assert.equal([...f.server.records.values()].filter(record => record.type === 'report').length, 2);
});

test('reminder reconciliation receives committed state, asks on enable, and denial never rolls back local edits', async t => {
  const calls = [];
  let denied = false;
  let throwing = false;
  const f = fixture(t, new Server(), { reconcileReminders: async (tasks, enabled, ask) => {
    calls.push({ tasks: structuredClone(tasks), enabled, ask });
    if (throwing) throw new Error('OS notification service failed');
    return denied && enabled ? 'Notifications are disabled.' : null;
  } });
  await f.store.initialize();
  denied = true;
  await f.store.updateSettings({ remindersEnabled: true });
  assert.equal(calls.at(-1).ask, true);
  assert.equal(f.store.getSnapshot().reminderError, 'Notifications are disabled.');
  assert.equal(f.store.getSnapshot().syncError, null);
  assert.equal(f.store.getSnapshot().syncState, 'idle');
  const task = await f.store.createTask({ title: 'durable reminder', remindAt: new Date(Date.now() + 60000).toISOString() });
  assert.equal(calls.at(-1).tasks[0].id, task.id);
  assert.equal(calls.at(-1).ask, false);
  throwing = true;
  await f.store.updateTask(task.id, { title: 'saved despite OS failure' });
  assert.match(f.store.getSnapshot().reminderError, /Could not schedule/);
  assert.equal(f.store.getSnapshot().syncState, 'idle');
  assert.equal(f.store.getSnapshot().syncError, null);
  throwing = false;
  await f.store.updateSettings({ remindersEnabled: false });
  assert.equal(f.store.getSnapshot().reminderError, null);
  assert.equal(f.store.getSnapshot().syncError, null);
  const reopened = f.open();
  await reopened.initialize();
  assert.equal(reopened.getSnapshot().tasks[0].title, 'saved despite OS failure');
});

test('remote tombstones keep local pending intent visible until explicit conflict resolution', async t => {
  const f = fixture(t);
  await f.connect();
  const task = await f.store.createTask({ title: 'shared' });
  await f.store.sync();
  await f.store.updateTask(task.id, { notes: 'offline edits must not disappear' });
  f.server.write('task', task.id, null);
  await f.store.sync();
  assert.equal(f.store.getSnapshot().tasks[0].notes, 'offline edits must not disappear');
  assert.equal(f.store.getSnapshot().conflicts[0].remote.deleted, true);
  await f.store.resolveConflict(f.store.getSnapshot().conflicts[0].mutation.id, 'remote');
  assert.equal(f.store.getSnapshot().tasks.length, 0);
});

// Build/run cmd/adhd-sync separately, then set these variables for exact HTTP interop.
test('real Go sync HTTP: two SQLite devices, lost acknowledgement, projects, reports and explicit conflict recovery', {
  skip: !process.env.MOBILE_SYNC_TEST_URL,
}, async t => {
  const url = process.env.MOBILE_SYNC_TEST_URL;
  const token = process.env.MOBILE_SYNC_TEST_TOKEN;
  assert.ok(token, 'MOBILE_SYNC_TEST_TOKEN is required for the private integration server');
  let loseAck = true;
  const pushSizes = [];
  const a = fixture(t, { token, fetch: async (requestUrl, options) => {
    if (options.method === 'POST') pushSizes.push(Buffer.byteLength(options.body));
    const response = await globalThis.fetch(requestUrl, options);
    if (loseAck && options.method === 'POST' && response.ok) { loseAck = false; await response.text(); throw new TypeError('Lost HTTP ack after real server committed'); }
    return response;
  } });
  const b = fixture(t, { token, fetch: globalThis.fetch });
  await a.store.updateSettings({ serverUrl: url }, token);
  await b.store.updateSettings({ serverUrl: url }, token);
  const project = await a.store.createProject('HTTP integration');
  const parent = await a.store.createTask({ title: 'real parent', projectId: project.id });
  const child = await a.store.createTask({ title: 'real child', parentId: parent.id });
  await a.store.sync();
  assert.equal(a.store.getSnapshot().syncState, 'offline');
  await a.store.sync();
  assert.equal(a.store.getSnapshot().pendingCount, 0);
  await b.store.sync();
  assert.equal(b.store.getSnapshot().tasks.find(task => task.id === child.id).parentId, parent.id);
  await a.store.updateTask(parent.id, { title: 'A offline title' });
  await b.store.updateTask(parent.id, { notes: 'B remote notes' });
  await b.store.sync();
  await a.store.sync();
  assert.equal(a.store.getSnapshot().conflicts.length, 1);
  await a.store.resolveConflict(a.store.getSnapshot().conflicts[0].mutation.id, 'local');
  await a.store.sync();
  assert.equal(a.store.getSnapshot().pendingCount, 0);
  await a.store.updateTask(parent.id, { status: 'done' });
  await a.store.sync();
  assert.equal(a.store.getSnapshot().pendingCount, 0);
  await b.store.sync();
  assert.equal(b.store.getSnapshot().tasks.find(task => task.id === parent.id).notes, 'B remote notes');
  assert.equal(b.store.getSnapshot().tasks.find(task => task.id === child.id).status, 'done');
  assert.equal(b.store.getSnapshot().reports.filter(report => report.taskId === child.id).length, 1);
  const preserved = a.store.getSnapshot();
  await a.store.updateSettings({ serverUrl: '' }, '');
  assert.deepEqual(a.store.getSnapshot().projects, preserved.projects);
  assert.deepEqual(a.store.getSnapshot().tasks, preserved.tasks);
  assert.deepEqual(a.store.getSnapshot().reports, preserved.reports);
  await a.store.updateTask(child.id, { notes: 'offline after disconnect' });
  await a.store.updateSettings({ serverUrl: url }, token);
  await a.store.sync();
  assert.equal(a.store.getSnapshot().pendingCount, 0);
  await a.store.deleteTask(parent.id);
  const lateChild = await b.store.createTask({ title: 'remote child created after offline parent deletion', parentId: parent.id });
  await b.store.sync();
  await a.store.sync();
  const parentConflict = a.store.getSnapshot().conflicts.find(item => item.mutation.entityId === parent.id);
  assert.ok(parentConflict, 'new remote child must expose parent conflict on real service');
  await a.store.resolveConflict(parentConflict.mutation.id, 'local');
  await a.store.sync();
  assert.equal(a.store.getSnapshot().pendingCount, 0);
  await b.store.sync();
  assert.equal(b.store.getSnapshot().tasks.some(task => task.id === parent.id), false);
  assert.equal(b.store.getSnapshot().tasks.find(task => task.id === child.id).parentId, null);
  assert.equal(b.store.getSnapshot().tasks.find(task => task.id === lateChild.id).parentId, null);
  for (let i = 0; i < 22; i++) await a.store.createTask({ title: `HTTP valid large ${i}`, notes: 'n'.repeat(50000) });
  for (let i = 0; i < 4; i++) await a.store.createTask({ title: `HTTP valid escaped ${i}`, notes: '\u0000'.repeat(10000) });
  const small = await a.store.createTask({ title: 'HTTP small independent edit' });
  await a.store.sync();
  assert.equal(a.store.getSnapshot().pendingCount, 0, a.store.getSnapshot().syncError ?? JSON.stringify(a.store.getSnapshot().conflicts.map(conflict => ({reason: conflict.reason, title: conflict.mutation.data?.title}))));
  assert.ok(pushSizes.every(bytes => bytes <= 1048576));
  await b.store.sync();
  assert.ok(b.store.getSnapshot().tasks.some(task => task.id === small.id));
});


test('database initialization failure is visible and an explicit retry recovers without a permanent loading state', async t => {
  const db = new SQLiteAdapter(':memory:');
  t.after(() => db.raw.close());
  let failed = false;
  const store = createAppStore({
    openDatabase: async () => { if (!failed) { failed = true; throw new Error('Device storage temporarily unavailable'); } return db; },
    uuid: randomUUID, getToken: async () => null, setToken: async () => {},
  });
  await assert.rejects(store.initialize(), /storage temporarily/);
  assert.equal(store.getSnapshot().ready, false);
  assert.equal(store.getSnapshot().syncState, 'error');
  assert.match(store.getSnapshot().syncError, /storage temporarily/);
  await store.initialize();
  assert.equal(store.getSnapshot().ready, true);
  assert.equal(store.getSnapshot().syncState, 'idle');
  assert.equal(store.getSnapshot().syncError, null);
  await store.createTask({ title: 'works after retry' });
  assert.equal(store.getSnapshot().pendingCount, 1);
});

test('disconnect keeps synced data and endpoint identity across reopen, offline edits and safe same-server reconnection', async t => {
  const f = fixture(t);
  await f.connect();
  const project = await f.store.createProject('kept project');
  const parent = await f.store.createTask({ title: 'kept parent', projectId: project.id });
  const child = await f.store.createTask({ title: 'kept child', parentId: parent.id });
  await f.store.updateTask(parent.id, { status: 'done' });
  await f.store.sync();
  const original = f.store.getSnapshot();
  await f.store.updateSettings({ serverUrl: '' }, '');
  assert.deepEqual(f.store.getSnapshot().tasks, original.tasks);
  assert.deepEqual(f.store.getSnapshot().projects, original.projects);
  assert.deepEqual(f.store.getSnapshot().reports, original.reports);
  assert.equal(f.tokens.get('http://sync.test'), '');
  await f.store.updateTask(child.id, { notes: 'edited while disconnected' });
  const reopened = f.open();
  await reopened.initialize();
  assert.equal(reopened.getSnapshot().settings.serverUrl, '');
  assert.equal(reopened.getSnapshot().tasks.find(task => task.id === child.id).notes, 'edited while disconnected');
  await assert.rejects(reopened.updateSettings({ serverUrl: 'https://other.test' }, 'different-token'), /migration|workspace|pending/i);
  assert.equal(reopened.getSnapshot().tasks.length, 2);
  await reopened.updateSettings({ serverUrl: 'http://sync.test' }, f.server.token);
  await reopened.sync();
  assert.equal(reopened.getSnapshot().pendingCount, 0);
  assert.equal(f.server.records.get(`task:${child.id}`).data.notes, 'edited while disconnected');
  await assert.rejects(reopened.updateSettings({ serverUrl: 'https://other.test' }, 'different-token'), /migration|workspace/i);
  assert.equal(reopened.getSnapshot().tasks.length, 2);
});

test('a newly discovered remote child turns parent deletion into an explicit conflict while unrelated edits sync', async t => {
  const f = fixture(t);
  await f.connect();
  const parent = await f.store.createTask({ title: 'parent' });
  await f.store.sync();
  await f.store.deleteTask(parent.id);
  const id = randomUUID();
  f.server.write('task', id, { ...parent, id, title: 'remote child', parentId: parent.id });
  const unrelated = await f.store.createTask({ title: 'unrelated still syncs' });
  await f.store.sync();
  await f.store.sync();
  assert.equal(f.store.getSnapshot().conflicts.length, 1);
  const conflict = f.store.getSnapshot().conflicts[0];
  assert.equal(conflict.mutation.entityId, parent.id);
  assert.equal(conflict.mutation.data, null, 'the original deletion intent is retained');
  assert.match(conflict.reason, /child|subtask/i);
  assert.ok(f.server.records.has(`task:${unrelated.id}`));
  await f.store.resolveConflict(conflict.mutation.id, 'local');
  await f.store.sync();
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.equal(f.server.records.get(`task:${parent.id}`).deleted, true);
  assert.equal(f.server.records.get(`task:${id}`).data.parentId, null);
});

test('choosing remote on a child detach conflict exposes the dependent parent deletion instead of silently stalling', async t => {
  const f = fixture(t);
  await f.connect();
  const parent = await f.store.createTask({ title: 'parent' });
  const child = await f.store.createTask({ title: 'child', parentId: parent.id });
  await f.store.sync();
  await f.store.deleteTask(parent.id);
  f.server.write('task', child.id, { ...f.server.records.get(`task:${child.id}`).data, notes: 'remote edit before detach' });
  await f.store.sync();
  const childConflict = f.store.getSnapshot().conflicts.find(item => item.mutation.entityId === child.id);
  assert.ok(childConflict);
  await f.store.resolveConflict(childConflict.mutation.id, 'remote');
  await f.store.sync();
  const parentConflict = f.store.getSnapshot().conflicts.find(item => item.mutation.entityId === parent.id);
  assert.ok(parentConflict, 'the parent deletion must become explicitly recoverable');
  await f.store.resolveConflict(parentConflict.mutation.id, 'remote');
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.ok(f.store.getSnapshot().tasks.some(task => task.id === parent.id));
  assert.equal(f.store.getSnapshot().tasks.find(task => task.id === child.id).notes, 'remote edit before detach');
});

test('push batches honor the one MiB UTF8 JSON envelope limit, including escaped fields, without blocking small edits', async t => {
  const server = new Server();
  const baseFetch = server.fetch.bind(server);
  const sizes = [];
  server.fetch = async (url, options) => {
    if (options.method === 'POST') {
      const bytes = Buffer.byteLength(options.body);
      sizes.push(bytes);
      if (bytes > 1048576) return Response.json({ error: 'request_too_large' }, { status: 413 });
    }
    return baseFetch(url, options);
  };
  const f = fixture(t, server);
  await f.connect();
  for (let i = 0; i < 22; i++) await f.store.createTask({ title: `large ${i}`, notes: 'x'.repeat(50000) });
  for (let i = 0; i < 4; i++) await f.store.createTask({ title: `escaped ${i}`, notes: '\u0000'.repeat(10000) });
  await f.store.createTask({ title: 'small independent edit', notes: '🚀'.repeat(10) });
  await f.store.sync();
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.ok(sizes.length > 1);
  assert.ok(sizes.every(bytes => bytes <= 1048576));
  assert.equal(server.records.size, 27);
});

test('field byte limits reject invalid new edits before transaction and preserve existing data/outbox', async t => {
  const f = fixture(t);
  const task = await f.store.createTask({ title: 'valid' });
  const pendingCount = f.store.getSnapshot().pendingCount;
  await assert.rejects(f.store.updateTask(task.id, { notes: '🚀'.repeat(12501) }), /notes.*50000/i);
  await assert.rejects(f.store.createTask({ title: 't'.repeat(2001) }), /title.*2000/i);
  await assert.rejects(f.store.createProject('名'.repeat(167)), /name.*500/i);
  await assert.rejects(f.store.createTask({ title: 'oversized escaped JSON', notes: '\u0000'.repeat(12000) }), /65536.*JSON/i);
  assert.equal(f.store.getSnapshot().tasks[0].notes, '');
  assert.equal(f.store.getSnapshot().pendingCount, pendingCount);
});

test('an invalid oversized legacy mutation becomes an individual conflict and does not freeze unrelated rows', async t => {
  const f = fixture(t);
  await f.connect();
  const bad = await f.store.createTask({ title: 'legacy invalid item' });
  const row = f.db.raw.prepare('SELECT * FROM outbox WHERE entity_id=?').get(bad.id);
  const mutation = JSON.parse(row.mutation);
  mutation.data.notes = 'n'.repeat(1100000);
  f.db.raw.prepare('UPDATE outbox SET mutation=?,intent=? WHERE id=?').run(JSON.stringify(mutation), JSON.stringify(mutation.data), row.id);
  const good = await f.store.createTask({ title: 'small valid item' });
  await f.store.sync();
  assert.ok(f.server.records.has(`task:${good.id}`));
  assert.equal(f.store.getSnapshot().conflicts.length, 1);
  assert.match(f.store.getSnapshot().conflicts[0].reason, /notes|size|large/i);
  assert.equal(f.db.raw.prepare('SELECT sent FROM outbox WHERE id=?').get(row.id).sent, 0);
});

test('byte-limited batches defer a child until its skipped parent create is selected or confirmed', async t => {
  const f = fixture(t);
  await f.connect();
  for (let i = 0; i < 20; i++) await f.store.createTask({ title: `batch filler ${i}`, notes: 'x'.repeat(50000) });
  const parent = await f.store.createTask({ title: 'large parent deferred to next batch', notes: 'y'.repeat(50000) });
  const child = await f.store.createTask({ title: 'small child must follow parent', parentId: parent.id });
  await f.store.sync();
  assert.equal(f.store.getSnapshot().pendingCount, 0);
  assert.equal(f.server.records.get(`task:${child.id}`).data.parentId, parent.id);
});
