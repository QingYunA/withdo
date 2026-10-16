import type {
  AppSnapshot, AppStore, Conflict, EntityData, EntityType, Mutation, NewTask,
  Project, PullResponse, PushResult, Report, Settings, SnapshotResponse, SyncRecord, Task, TaskPatch,
} from '../model';

export interface SqlExecutor {
  execAsync(sql: string): Promise<void>;
  runAsync(sql: string, ...params: (string | number | null)[]): Promise<unknown>;
  getAllAsync<T>(sql: string, ...params: (string | number | null)[]): Promise<T[]>;
}
export interface Database extends SqlExecutor {
  withExclusiveTransactionAsync(work: (tx: SqlExecutor) => Promise<void>): Promise<void>;
}
export interface StoreDependencies {
  openDatabase(): Promise<Database>;
  uuid(): string;
  now?: () => Date;
  fetch?: typeof globalThis.fetch;
  getToken(serverUrl: string): Promise<string | null>;
  setToken(serverUrl: string, token: string): Promise<void>;
  reconcileReminders?: (tasks: Task[], enabled: boolean, requestPermission: boolean) => Promise<string | null>;
  autoSync?: boolean;
}
interface PendingRow {
  seq: number;
  id: string;
  entity_type: EntityType;
  entity_id: string;
  mutation: string;
  intent: string;
  sent: number;
  status: 'pending' | 'conflict';
  remote: string | null;
  reason: string | null;
}
interface ConfirmedRow { type: EntityType; id: string; version: number; deleted: number; data: string | null }
type Intent = Partial<EntityData> | null;
const defaults: Settings = { serverUrl: '', language: 'system', appearance: 'system', remindersEnabled: false };
const schema = `
  PRAGMA journal_mode = WAL;
  PRAGMA foreign_keys = ON;
  CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL);
  CREATE TABLE IF NOT EXISTS confirmed (
    type TEXT NOT NULL, id TEXT NOT NULL, version INTEGER NOT NULL,
    deleted INTEGER NOT NULL, data TEXT, PRIMARY KEY(type,id)
  );
  CREATE TABLE IF NOT EXISTS outbox (
    seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT NOT NULL UNIQUE,
    entity_type TEXT NOT NULL, entity_id TEXT NOT NULL,
    mutation TEXT NOT NULL, intent TEXT NOT NULL, sent INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending', remote TEXT, reason TEXT
  );
  CREATE INDEX IF NOT EXISTS outbox_entity ON outbox(entity_type,entity_id,seq);
  CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, data TEXT NOT NULL);
  CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY, data TEXT NOT NULL,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
    parent_id TEXT REFERENCES tasks(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED
  );
  CREATE TABLE IF NOT EXISTS reports (
    id TEXT PRIMARY KEY, data TEXT NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED
  );
  PRAGMA user_version = 1;
`;
const key = (type: EntityType, id: string) => `${type}:${id}`;
const parse = <T>(value: string): T => JSON.parse(value) as T;
const terminal = (status: Task['status']) => status === 'done' || status === 'cancelled';

export function createAppStore(deps: StoreDependencies): AppStore {
  let db: Database;
  let initialization: Promise<void> | null = null;
  let serial: Promise<unknown> = Promise.resolve();
  let syncing: Promise<void> | null = null;
  let localRevision = 0;
  let syncTimer: ReturnType<typeof setTimeout> | undefined;
  let snapshot: AppSnapshot = {
    ready: false, projects: [], tasks: [], reports: [], settings: { ...defaults }, pendingCount: 0,
    conflicts: [], syncState: 'idle', syncError: null, reminderError: null, lastSyncedAt: null,
  };
  const listeners = new Set<() => void>();
  const now = () => (deps.now?.() ?? new Date()).toISOString();
  const publish = (patch: Partial<AppSnapshot>) => {
    snapshot = { ...snapshot, ...patch };
    listeners.forEach(listener => listener());
  };
  const locked = <T>(work: () => Promise<T>): Promise<T> => {
    const result = serial.then(work);
    serial = result.catch(() => undefined);
    return result;
  };
  const getMeta = async (tx: SqlExecutor, name: string) =>
    (await tx.getAllAsync<{ value: string }>('SELECT value FROM metadata WHERE key = ?', name))[0]?.value ?? null;
  const setMeta = (tx: SqlExecutor, name: string, value: string) =>
    tx.runAsync('INSERT INTO metadata(key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value', name, value);
  const pendingRows = (tx: SqlExecutor) => tx.getAllAsync<PendingRow>('SELECT * FROM outbox ORDER BY seq');
  const putRecord = async (tx: SqlExecutor, record: SyncRecord) => {
    await tx.runAsync(`INSERT INTO confirmed(type,id,version,deleted,data) VALUES (?,?,?,?,?)
      ON CONFLICT(type,id) DO UPDATE SET version=excluded.version,deleted=excluded.deleted,data=excluded.data
      WHERE excluded.version >= confirmed.version`, record.type, record.id, record.version, record.deleted ? 1 : 0,
    record.data ? JSON.stringify(record.data) : null);
  };
  const confirmedRecord = async (tx: SqlExecutor, type: EntityType, id: string): Promise<SyncRecord | null> => {
    const row = (await tx.getAllAsync<ConfirmedRow>('SELECT * FROM confirmed WHERE type=? AND id=?', type, id))[0];
    return row ? { type, id, version: row.version, deleted: !!row.deleted, data: row.data ? parse<EntityData>(row.data) : null } : null;
  };
  const apply = (current: EntityData | undefined, row: PendingRow): EntityData | null => {
    const intent = parse<Intent>(row.intent);
    if (intent === null) return null;
    const original = parse<Mutation>(row.mutation).data;
    return { ...original, ...current, ...intent } as EntityData;
  };
  const materialize = async (tx: SqlExecutor) => {
    for (const conflict of (await pendingRows(tx)).filter(row => row.status === 'conflict')) {
      const remote = await confirmedRecord(tx, conflict.entity_type, conflict.entity_id);
      await tx.runAsync('UPDATE outbox SET remote=? WHERE id=?', remote ? JSON.stringify(remote) : null, conflict.id);
    }
    const entities = new Map<string, EntityData>();
    for (const row of await tx.getAllAsync<ConfirmedRow>('SELECT * FROM confirmed WHERE deleted=0')) {
      if (row.data) entities.set(key(row.type, row.id), parse<EntityData>(row.data));
    }
    for (const row of await pendingRows(tx)) {
      const id = key(row.entity_type, row.entity_id);
      const data = apply(entities.get(id), row);
      if (data) entities.set(id, data); else entities.delete(id);
    }
    const projects: Project[] = [], tasks: Task[] = [], reports: Report[] = [];
    for (const [id, data] of entities) {
      if (id.startsWith('project:')) projects.push(data as Project);
      if (id.startsWith('task:')) tasks.push(data as Task);
      if (id.startsWith('report:')) reports.push(data as Report);
    }
    const taskMap = new Map(tasks.map(task => [task.id, task]));
    for (const task of tasks) {
      if (task.projectId && !entities.has(key('project', task.projectId))) task.projectId = null;
      let ancestor = task.parentId;
      const visited = new Set([task.id]);
      while (ancestor) {
        if (visited.has(ancestor) || !taskMap.has(ancestor)) { task.parentId = null; break; }
        visited.add(ancestor);
        ancestor = taskMap.get(ancestor)?.parentId ?? null;
      }
    }
    await tx.execAsync('DELETE FROM reports; DELETE FROM tasks; DELETE FROM projects;');
    for (const item of projects) await tx.runAsync('INSERT INTO projects(id,data) VALUES (?,?)', item.id, JSON.stringify(item));
    for (const item of tasks) await tx.runAsync('INSERT INTO tasks(id,data,project_id,parent_id) VALUES (?,?,?,?)', item.id, JSON.stringify(item), item.projectId, item.parentId);
    for (const item of reports) if (taskMap.has(item.taskId)) await tx.runAsync('INSERT INTO reports(id,data,task_id) VALUES (?,?,?)', item.id, JSON.stringify(item), item.taskId);
  };
  const refresh = async () => {
    const [projects, tasks, reports, pending, settings, lastSyncedAt] = await Promise.all([
      db.getAllAsync<{ data: string }>('SELECT data FROM projects ORDER BY id'),
      db.getAllAsync<{ data: string }>('SELECT data FROM tasks ORDER BY id'),
      db.getAllAsync<{ data: string }>('SELECT data FROM reports ORDER BY id'),
      pendingRows(db), getMeta(db, 'settings'), getMeta(db, 'lastSyncedAt'),
    ]);
    const conflicts: Conflict[] = pending.filter(row => row.status === 'conflict').map(row => ({
      mutation: parse<Mutation>(row.mutation), remote: row.remote ? parse<SyncRecord>(row.remote) : null,
      reason: row.reason ?? 'Conflict',
    }));
    publish({
      ready: true, projects: projects.map(row => parse<Project>(row.data)), tasks: tasks.map(row => parse<Task>(row.data)),
      reports: reports.map(row => parse<Report>(row.data)), settings: settings ? parse<Settings>(settings) : { ...defaults },
      pendingCount: pending.length, conflicts, lastSyncedAt,
    });
  };
  const reminders = async (ask = false) => {
    if (!deps.reconcileReminders) return;
    try {
      const warning = await deps.reconcileReminders(snapshot.tasks, snapshot.settings.remindersEnabled, ask);
      publish({ reminderError: warning });
    } catch {
      publish({ reminderError: 'Could not schedule reminders. Check notification permissions in system settings.' });
    }
  };
  const autoSync = () => {
    if (!deps.autoSync || !snapshot.settings.serverUrl) return;
    clearTimeout(syncTimer);
    syncTimer = setTimeout(() => { void store.sync(); }, 400);
  };
  const transact = async (work: (tx: SqlExecutor) => Promise<void>) => {
    await store.initialize();
    await locked(async () => {
      await db.withExclusiveTransactionAsync(async tx => { await work(tx); await materialize(tx); });
      await refresh();
      localRevision++;
    });
    await reminders();
    autoSync();
  };
  const queue = async (tx: SqlExecutor, type: EntityType, id: string, data: EntityData | null, intent: Intent) => {
    const version = (await confirmedRecord(tx, type, id))?.version ?? 0;
    const mutation: Mutation = { id: deps.uuid(), type, entityId: id, baseVersion: version, data };
    await tx.runAsync('INSERT INTO outbox(id,entity_type,entity_id,mutation,intent) VALUES (?,?,?,?,?)',
      mutation.id, type, id, JSON.stringify(mutation), JSON.stringify(intent));
  };
  const requireTask = (id: string) => {
    const task = snapshot.tasks.find(item => item.id === id);
    if (!task) throw new Error('Task no longer exists');
    return task;
  };
  const validateTask = (task: Task) => {
    if (!task.title.trim()) throw new Error('Task title is required');
    if (task.projectId && !snapshot.projects.some(item => item.id === task.projectId)) throw new Error('Project no longer exists');
    for (const value of [task.dueAt, task.remindAt]) if (value && !Number.isFinite(Date.parse(value))) throw new Error('Invalid date');
    let parentId = task.parentId;
    const seen = new Set([task.id]);
    while (parentId) {
      if (seen.has(parentId)) throw new Error('A task cannot be its own ancestor');
      seen.add(parentId);
      const parent = requireTask(parentId);
      if (parent.projectId !== task.projectId) throw new Error('Subtasks must belong to the same project as their parent.');
      parentId = parent.parentId;
    }
    if (snapshot.tasks.some(child => child.parentId === task.id && child.projectId !== task.projectId)) throw new Error('Move or detach subtasks before changing their parent project.');
  };
  const rebaseNext = async (tx: SqlExecutor, type: EntityType, id: string) => {
    const next = (await pendingRows(tx)).find(row => row.entity_type === type && row.entity_id === id);
    if (!next || next.sent || next.status === 'conflict') return;
    const record = await confirmedRecord(tx, type, id);
    const mutation = parse<Mutation>(next.mutation);
    mutation.baseVersion = record?.version ?? 0;
    mutation.data = apply(record?.data ?? undefined, next);
    await tx.runAsync('UPDATE outbox SET mutation=? WHERE id=?', JSON.stringify(mutation), next.id);
  };
  const request = async <T>(server: string, token: string, path: string, body?: unknown): Promise<T> => {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 15000);
    try {
      const response = await (deps.fetch ?? globalThis.fetch)(`${server}${path}`, {
        method: body === undefined ? 'GET' : 'POST', signal: controller.signal,
        headers: { Authorization: `Bearer ${token}`, ...(body === undefined ? {} : { 'Content-Type': 'application/json' }) },
        ...(body === undefined ? {} : { body: JSON.stringify(body) }),
      });
      if (!response.ok) throw new SyncError(response.status, response.status === 401 || response.status === 403
        ? 'Server rejected the token. Update the sync token in settings.' : `Sync server returned HTTP ${response.status}`);
      return await response.json() as T;
    } finally { clearTimeout(timeout); }
  };
  const replaceSnapshot = async (server: string, token: string) => {
    const incoming = await request<SnapshotResponse>(server, token, '/v1/sync/snapshot');
    if (!incoming.epoch || !Number.isSafeInteger(incoming.cursor) || !Array.isArray(incoming.records)) throw new Error('Invalid server snapshot');
    await locked(async () => {
      await db.withExclusiveTransactionAsync(async tx => {
        await tx.execAsync('DELETE FROM confirmed');
        for (const record of incoming.records) await putRecord(tx, record);
        await setMeta(tx, 'epoch', incoming.epoch);
        await setMeta(tx, 'cursor', String(incoming.cursor));
        await materialize(tx);
      });
      await refresh();
    });
  };
  const pull = async (server: string, token: string) => {
    if (!(await getMeta(db, 'epoch'))) await replaceSnapshot(server, token);
    for (let page = 0; page < 100; page++) {
      const cursor = Number(await getMeta(db, 'cursor') ?? 0);
      const epoch = await getMeta(db, 'epoch') ?? '';
      let incoming: PullResponse;
      try { incoming = await request<PullResponse>(server, token, `/v1/sync/pull?cursor=${cursor}&limit=200&epoch=${encodeURIComponent(epoch)}`); }
      catch (error) {
        if (error instanceof SyncError && error.status === 409) { await replaceSnapshot(server, token); continue; }
        throw error;
      }
      if (incoming.epoch !== epoch) { await replaceSnapshot(server, token); continue; }
      if (!Number.isSafeInteger(incoming.cursor) || incoming.cursor < cursor || !Array.isArray(incoming.changes)
        || (incoming.hasMore && incoming.cursor === cursor)) throw new Error('Invalid server cursor');
      await locked(async () => {
        await db.withExclusiveTransactionAsync(async tx => {
          for (const change of incoming.changes) await putRecord(tx, change.record);
          await setMeta(tx, 'cursor', String(incoming.cursor));
          await materialize(tx);
        });
        await refresh();
      });
      if (!incoming.hasMore) return;
    }
    throw new Error('Sync has more changes; sync again to continue.');
  };
  const doSync = async () => {
    await store.initialize();
    const server = snapshot.settings.serverUrl;
    if (!server) return;
    publish({ syncState: 'syncing', syncError: null });
    try {
      const token = await deps.getToken(server);
      if (!token) throw new Error('Add a server token in settings to enable sync.');
      await pull(server, token);
      const deviceId = await getMeta(db, 'deviceId');
      for (let round = 0; round < 100; round++) {
        const outgoing = await locked(async () => {
          const mutations: Mutation[] = [];
          await db.withExclusiveTransactionAsync(async tx => {
            const seen = new Set<string>();
            const pending = await pendingRows(tx);
            const confirmedTasks = await tx.getAllAsync<ConfirmedRow>("SELECT * FROM confirmed WHERE type='task' AND deleted=0");
            for (const row of pending) {
              const entity = key(row.entity_type, row.entity_id);
              if (seen.has(entity)) continue;
              seen.add(entity);
              if (row.status === 'conflict') continue;
              const mutation = parse<Mutation>(row.mutation);
              // A parent tombstone must follow all child detaches, including older queued child creates.
              if (mutation.type === 'task' && mutation.data === null && (
                confirmedTasks.some(item => item.data && parse<Task>(item.data).parentId === mutation.entityId)
                || pending.some(item => item.entity_type === 'task' && (parse<Mutation>(item.mutation).data as Task | null)?.parentId === mutation.entityId)
              )) continue;
              await tx.runAsync('UPDATE outbox SET sent=1 WHERE id=?', row.id);
              mutations.push(mutation);
              if (mutations.length === 50) break;
            }
          });
          return mutations;
        });
        if (!outgoing.length) break;
        const response = await request<{ results: PushResult[] }>(server, token, '/v1/sync/push', { deviceId, mutations: outgoing });
        if (!Array.isArray(response.results) || outgoing.some(item => !response.results.some(result => result.mutationId === item.id))) throw new Error('Incomplete push response; pending changes will be retried.');
        await locked(async () => {
          await db.withExclusiveTransactionAsync(async tx => {
            for (const mutation of outgoing) {
              const result = response.results.find(item => item.mutationId === mutation.id)!;
              if (result.status === 'accepted') {
                if (!result.record) throw new Error('Accepted mutation has no canonical record');
                await putRecord(tx, result.record);
                await tx.runAsync('DELETE FROM outbox WHERE id=?', mutation.id);
                await rebaseNext(tx, mutation.type, mutation.entityId);
              } else {
                if (result.record) await putRecord(tx, result.record);
                await tx.runAsync('UPDATE outbox SET status=?,remote=?,reason=? WHERE id=?', 'conflict',
                  result.record ? JSON.stringify(result.record) : null, result.error ?? (result.status === 'conflict' ? 'Another device changed this item.' : 'Server rejected this change.'), mutation.id);
              }
            }
            await materialize(tx);
          });
          await refresh();
        });
      }
      await pull(server, token);
      await locked(async () => { await setMeta(db, 'lastSyncedAt', now()); await refresh(); });
      publish({ syncState: 'idle', syncError: null });
      await reminders();
    } catch (error) {
      publish({ syncState: error instanceof TypeError || (error instanceof Error && error.name === 'AbortError') ? 'offline' : 'error',
        syncError: error instanceof Error ? error.message : 'Sync failed. Local changes are saved.' });
    }
  };
  const store: AppStore = {
    getSnapshot: () => snapshot,
    subscribe(listener) { listeners.add(listener); return () => { listeners.delete(listener); }; },
    initialize() {
      if (!initialization) initialization = locked(async () => {
        db = await deps.openDatabase();
        const version = (await db.getAllAsync<{ user_version: number }>('PRAGMA user_version'))[0]?.user_version ?? 0;
        if (version > 1) throw new Error('This database requires a newer application version.');
        await db.execAsync(schema);
        await db.withExclusiveTransactionAsync(async tx => {
          if (!(await getMeta(tx, 'deviceId'))) await setMeta(tx, 'deviceId', deps.uuid());
          await materialize(tx);
        });
        await refresh();
        publish({ syncState: 'idle', syncError: null });
        await reminders();
        autoSync();
      }).catch(error => {
        initialization = null;
        publish({ syncState: 'error', syncError: error instanceof Error ? error.message : 'Could not open local database.' });
        throw error;
      });
      return initialization;
    },
    async createTask(input: NewTask) {
      let task!: Task;
      await transact(async tx => {
        const timestamp = now();
        task = { id: deps.uuid(), projectId: input.projectId === undefined && input.parentId ? requireTask(input.parentId).projectId : input.projectId ?? null, parentId: input.parentId ?? null,
          title: input.title.trim(), notes: input.notes ?? '', status: input.status ?? 'todo', priority: input.priority ?? 'P2',
          source: 'user', dueAt: input.dueAt ?? null, remindAt: input.remindAt ?? null, branch: '', prUrl: '',
          createdAt: timestamp, updatedAt: timestamp, completedAt: terminal(input.status ?? 'todo') ? timestamp : null };
        validateTask(task);
        await queue(tx, 'task', task.id, task, task);
      });
      return task;
    },
    async updateTask(id: string, patch: TaskPatch) {
      await transact(async tx => {
        const existing = requireTask(id);
        const task: Task = { ...existing, ...patch, title: (patch.title ?? existing.title).trim(), updatedAt: now() };
        if (patch.status !== undefined) task.completedAt = terminal(patch.status) ? existing.completedAt ?? task.updatedAt : null;
        validateTask(task);
        const intent = { ...patch, updatedAt: task.updatedAt, ...(patch.title === undefined ? {} : { title: task.title }),
          ...(patch.status === undefined ? {} : { completedAt: task.completedAt }) };
        await queue(tx, 'task', id, task, intent);
        if (!terminal(existing.status) && terminal(task.status)) {
          const descendants = new Set([id]);
          let changed = true;
          while (changed) {
            changed = false;
            for (const candidate of snapshot.tasks) if (candidate.parentId && descendants.has(candidate.parentId) && !descendants.has(candidate.id)) {
              descendants.add(candidate.id); changed = true;
            }
          }
          for (const child of snapshot.tasks) if (child.id !== id && descendants.has(child.id) && !terminal(child.status)) {
            const closed = { status: 'done' as const, updatedAt: task.updatedAt, completedAt: task.updatedAt };
            await queue(tx, 'task', child.id, { ...child, ...closed }, closed);
            const report: Report = { id: deps.uuid(), taskId: child.id, agent: 'adhd-todo', status: 'done',
              summary: 'Closed with its parent task', nextStep: '', createdAt: task.updatedAt };
            await queue(tx, 'report', report.id, report, report);
          }
        }
      });
    },
    async deleteTask(id: string) {
      await transact(async tx => {
        requireTask(id);
        // Match desktop semantics: subtasks survive as top-level tasks.
        for (const child of snapshot.tasks.filter(task => task.parentId === id)) {
          const patch = { parentId: null, updatedAt: now() };
          await queue(tx, 'task', child.id, { ...child, ...patch }, patch);
        }
        await queue(tx, 'task', id, null, null);
      });
    },
    async createProject(name: string) {
      let project!: Project;
      await transact(async tx => {
        if (!name.trim()) throw new Error('Project name is required');
        project = { id: deps.uuid(), name: name.trim(), repoSlug: '', archivedAt: null, createdAt: now(), updatedAt: now() };
        await queue(tx, 'project', project.id, project, project);
      });
      return project;
    },
    async updateProject(id, patch) {
      await transact(async tx => {
        const old = snapshot.projects.find(project => project.id === id);
        if (!old) throw new Error('Project no longer exists');
        const project = { ...old, ...patch, name: (patch.name ?? old.name).trim(), updatedAt: now() };
        if (!project.name) throw new Error('Project name is required');
        await queue(tx, 'project', id, project, { ...patch, ...(patch.name === undefined ? {} : { name: project.name }), updatedAt: project.updatedAt });
      });
    },
    async updateSettings(patch, token) {
      await store.initialize();
      if (syncing && (patch.serverUrl !== undefined || token !== undefined)) throw new Error('Wait for the current sync to finish before changing the server connection.');
      await locked(async () => {
        const settings = { ...snapshot.settings, ...patch };
        settings.serverUrl = settings.serverUrl.trim().replace(/\/+$/, '');
        if (settings.serverUrl) {
          const url = new URL(settings.serverUrl);
          if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash) throw new Error('Use an HTTP or HTTPS server URL without credentials or query parameters.');
        }
        const changed = settings.serverUrl !== snapshot.settings.serverUrl;
        if (changed && snapshot.settings.serverUrl && snapshot.pendingCount) throw new Error('Sync or resolve all pending changes before changing the server. Your local changes are preserved.');
        if (token !== undefined && settings.serverUrl) await deps.setToken(settings.serverUrl, token.trim());
        await db.withExclusiveTransactionAsync(async tx => {
          await setMeta(tx, 'settings', JSON.stringify(settings));
          if (changed) {
            await tx.execAsync("DELETE FROM metadata WHERE key IN ('epoch','cursor','lastSyncedAt'); DELETE FROM confirmed;");
            await materialize(tx);
          }
        });
        await refresh();
      });
      await reminders(patch.remindersEnabled === true);
      autoSync();
    },
    sync() {
      if (!syncing) {
        const revision = localRevision;
        syncing = doSync().finally(() => {
          syncing = null;
          if (revision !== localRevision) autoSync();
        });
      }
      return syncing;
    },
    async resolveConflict(mutationId, resolution) {
      await transact(async tx => {
        const row = (await pendingRows(tx)).find(item => item.id === mutationId && item.status === 'conflict');
        if (!row) throw new Error('Conflict has already been resolved.');
        if (resolution === 'remote') await tx.runAsync('DELETE FROM outbox WHERE id=?', mutationId);
        else {
          const record = await confirmedRecord(tx, row.entity_type, row.entity_id);
          const mutation = parse<Mutation>(row.mutation);
          mutation.id = deps.uuid();
          mutation.baseVersion = record?.version ?? 0;
          mutation.data = apply(record?.data ?? undefined, row);
          await tx.runAsync("UPDATE outbox SET id=?,mutation=?,sent=0,status='pending',remote=NULL,reason=NULL WHERE id=?", mutation.id, JSON.stringify(mutation), mutationId);
        }
        await rebaseNext(tx, row.entity_type, row.entity_id);
      });
    },
  };
  return store;
}

class SyncError extends Error {
  readonly status: number;
  constructor(status: number, message: string) { super(message); this.status = status; }
}
