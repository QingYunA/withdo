import Svg, { Path, Rect } from 'react-native-svg';
import type { TaskStatus } from '../model';
import { iconPaths } from './icon-paths';
import { useUI } from './theme';

export type IconName = keyof typeof iconPaths;
export function Icon({
  name,
  size = 24,
  color,
}: {
  name: IconName;
  size?: number;
  color?: string;
}) {
  const { colors } = useUI();
  return (
    <Svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
    >
      {iconPaths[name].map((d, i) => (
        <Path key={i} d={d} fill={color ?? colors.secondary} />
      ))}
    </Svg>
  );
}
const ring = [
  '....####....',
  '..##....##..',
  '.#........#.',
  '.#........#.',
  '#..........#',
  '#..........#',
  '#..........#',
  '#..........#',
  '.#........#.',
  '.#........#.',
  '..##....##..',
  '....####....',
];
const disc = [
  '....####....',
  '..########..',
  '.##########.',
  '.##########.',
  '############',
  '############',
  '############',
  '############',
  '.##########.',
  '.##########.',
  '..########..',
  '....####....',
];
const marks: Partial<Record<TaskStatus, number[][]>> = {
  doing: [
    [6, 3],
    [7, 3],
    [6, 4],
    [7, 4],
    [8, 4],
    [6, 5],
    [7, 5],
    [8, 5],
    [6, 6],
    [7, 6],
    [8, 6],
    [6, 7],
    [7, 7],
    [8, 7],
    [6, 8],
    [7, 8],
  ],
  review: [
    [5, 4],
    [6, 4],
    [4, 5],
    [5, 5],
    [6, 5],
    [7, 5],
    [4, 6],
    [5, 6],
    [6, 6],
    [7, 6],
    [5, 7],
    [6, 7],
  ],
  done: [
    [3, 6],
    [3, 7],
    [4, 7],
    [4, 8],
    [5, 8],
    [5, 9],
    [6, 7],
    [6, 8],
    [7, 6],
    [7, 7],
    [8, 5],
    [8, 6],
    [9, 4],
    [9, 5],
  ],
  blocked: [
    [5, 2],
    [6, 2],
    [5, 3],
    [6, 3],
    [5, 4],
    [6, 4],
    [5, 5],
    [6, 5],
    [5, 6],
    [6, 6],
    [5, 8],
    [6, 8],
    [5, 9],
    [6, 9],
  ],
  cancelled: [
    [3, 8],
    [4, 7],
    [5, 6],
    [6, 5],
    [7, 4],
    [8, 3],
  ],
};
export function StatusIcon({
  status,
  size = 24,
}: {
  status: TaskStatus;
  size?: number;
}) {
  const { colors } = useUI();
  const filled = status === 'done' || status === 'blocked';
  const pixels: { x: number; y: number; fill: string }[] = [];
  let n = 0;
  (filled ? disc : ring).forEach((row, y) =>
    [...row].forEach((cell, x) => {
      if (cell !== '#') return;
      if (status !== 'backlog' || n % 3 !== 2)
        pixels.push({ x, y, fill: colors[status] });
      n++;
    }),
  );
  marks[status]?.forEach(([x, y]) =>
    pixels.push({ x, y, fill: filled ? '#FFFFFF' : colors[status] }),
  );
  return (
    <Svg
      width={size}
      height={size}
      viewBox="0 0 12 12"
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
    >
      {pixels.map((p, i) => (
        <Rect key={i} x={p.x} y={p.y} width={1} height={1} fill={p.fill} />
      ))}
    </Svg>
  );
}
