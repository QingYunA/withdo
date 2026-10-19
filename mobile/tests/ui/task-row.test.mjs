import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';
import ts from 'typescript';

// Exercise the actual rendered glyph's handler without loading Android native modules.
const source = readFileSync(
  new URL('../../src/ui/TaskRow.tsx', import.meta.url),
  'utf8',
);
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
}).outputText;
function renderGlyph(task, agent) {
  const mutations = [],
    navigations = [];
  const snapshot = { projects: [], reports: [] };
  const ui = {
    snapshot,
    now: Date.parse('2026-10-16T12:00:00Z'),
    locale: 'en',
    colors: {},
    t: (key) =>
      ({ openTask: 'Open task', complete: 'Mark complete' })[key] ?? key,
  };
  const dependencies = {
    'react/jsx-runtime': {
      jsx: (type, props) => ({ type, props }),
      jsxs: (type, props) => ({ type, props }),
    },
    'react-native': { View: 'View', Pressable: 'Pressable', Text: 'Text' },
    'expo-router': { router: { push: (route) => navigations.push(route) } },
    '../data': {
      appStore: {
        updateTask: async (id, patch) => mutations.push({ id, patch }),
      },
    },
    '../i18n': {
      dateLabel: () => '',
      statusKey: (status) => `status_${status}`,
    },
    './icons': { StatusIcon: 'StatusIcon' },
    './theme': { type: {}, useUI: () => ui },
    './components': {
      ErrorMessage: 'ErrorMessage',
      useAction: () => ({
        busy: false,
        error: null,
        run: (action) => action(),
      }),
    },
  };
  const exports = {};
  runInNewContext(compiled, {
    exports,
    require: (name) => {
      assert.ok(name in dependencies, `Unexpected dependency: ${name}`);
      return dependencies[name];
    },
  });
  const tree = exports.TaskRow({
    task,
    ...(agent === undefined ? {} : { agent }),
  });
  const glyph = tree.props.children[0].props.children[0];
  assert.equal(glyph.type, 'Pressable');
  return { glyph, mutations, navigations };
}
const statuses = [
  'backlog',
  'todo',
  'doing',
  'blocked',
  'review',
  'done',
  'cancelled',
];
for (const status of statuses) {
  test(`agent ${status} glyph opens details even when its agent prop is omitted`, () => {
    const { glyph, mutations, navigations } = renderGlyph({
      id: 'agent-child',
      source: 'agent',
      status,
      title: 'Agent subtask',
    });
    assert.equal(glyph.props.accessibilityRole, 'button');
    assert.equal(glyph.props.accessibilityLabel, 'Open task: Agent subtask');
    assert.equal(glyph.props.accessibilityState.checked, undefined);
    glyph.props.onPress();
    assert.equal(mutations.length, 0, 'agent glyph must never complete work');
    assert.equal(navigations.length, 1);
    assert.equal(navigations[0].pathname, '/task/[id]');
    assert.equal(navigations[0].params.id, 'agent-child');
  });
}
for (const status of statuses) {
  test(`personal ${status} glyph follows completion safety rules`, () => {
    const { glyph, mutations, navigations } = renderGlyph({
      id: 'personal',
      source: 'user',
      status,
      title: 'My task',
    });
    const terminal = status === 'done' || status === 'cancelled';
    assert.equal(
      glyph.props.accessibilityRole,
      terminal ? 'button' : 'checkbox',
    );
    assert.equal(
      glyph.props.accessibilityLabel,
      `${terminal ? 'Open task' : 'Mark complete'}: My task`,
    );
    glyph.props.onPress();
    assert.equal(navigations.length, terminal ? 1 : 0);
    assert.equal(mutations.length, terminal ? 0 : 1);
    if (!terminal) {
      assert.equal(glyph.props.accessibilityState.checked, false);
      assert.equal(mutations[0].id, 'personal');
      assert.equal(mutations[0].patch.status, 'done');
    }
  });
}
