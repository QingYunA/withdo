export type TaskStatus = 'backlog' | 'todo' | 'doing' | 'blocked' | 'review' | 'done' | 'cancelled';
export type Priority = 'P0' | 'P1' | 'P2' | 'P3';
export type EntityType = 'project' | 'task' | 'report';

export interface Project {
  id: string;
  name: string;
  repoSlug: string;
  archivedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface Task {
  id: string;
  projectId: string | null;
  parentId: string | null;
  title: string;
  notes: string;
  status: TaskStatus;
  priority: Priority;
  source: 'user' | 'agent';
  dueAt: string | null;
  remindAt: string | null;
  branch: string;
  prUrl: string;
  createdAt: string;
  updatedAt: string;
  completedAt: string | null;
}

export interface Report {
  id: string;
  taskId: string;
  agent: string;
  status: TaskStatus | null;
  summary: string;
  nextStep: string;
  createdAt: string;
}

export type EntityData = Project | Task | Report;
export interface SyncRecord {
  type: EntityType;
  id: string;
  version: number;
  deleted: boolean;
  data: EntityData | null;
}
export interface Mutation {
  id: string;
  type: EntityType;
  entityId: string;
  baseVersion: number;
  data: EntityData | null;
}
export interface PushResult {
  mutationId: string;
  status: 'accepted' | 'conflict' | 'rejected';
  record: SyncRecord | null;
  error?: string;
}
export interface PullResponse {
  epoch: string;
  cursor: number;
  changes: { seq: number; record: SyncRecord }[];
  hasMore: boolean;
}
export interface SnapshotResponse {
  epoch: string;
  cursor: number;
  records: SyncRecord[];
}

export type Language = 'system' | 'zh' | 'en';
export type Appearance = 'system' | 'light' | 'dark';
export interface Settings {
  serverUrl: string;
  language: Language;
  appearance: Appearance;
  remindersEnabled: boolean;
}
export interface Conflict {
  mutation: Mutation;
  remote: SyncRecord | null;
  reason: string;
}
export interface AppSnapshot {
  ready: boolean;
  projects: Project[];
  tasks: Task[];
  reports: Report[];
  settings: Settings;
  pendingCount: number;
  conflicts: Conflict[];
  syncState: 'idle' | 'syncing' | 'offline' | 'error';
  syncError: string | null;
  reminderError: string | null;
  lastSyncedAt: string | null;
}

export type NewTask = Pick<Task, 'title'> & Partial<Pick<Task, 'projectId' | 'parentId' | 'notes' | 'status' | 'priority' | 'dueAt' | 'remindAt'>>;
export type TaskPatch = Partial<Pick<Task, 'title' | 'notes' | 'status' | 'priority' | 'projectId' | 'parentId' | 'dueAt' | 'remindAt'>>;

export interface AppStore {
  getSnapshot(): AppSnapshot;
  subscribe(listener: () => void): () => void;
  initialize(): Promise<void>;
  createTask(input: NewTask): Promise<Task>;
  updateTask(id: string, patch: TaskPatch): Promise<void>;
  deleteTask(id: string): Promise<void>;
  createProject(name: string): Promise<Project>;
  updateProject(id: string, patch: Partial<Pick<Project, 'name' | 'archivedAt'>>): Promise<void>;
  updateSettings(patch: Partial<Settings>, token?: string): Promise<void>;
  sync(): Promise<void>;
  resolveConflict(mutationId: string, resolution: 'local' | 'remote'): Promise<void>;
}
