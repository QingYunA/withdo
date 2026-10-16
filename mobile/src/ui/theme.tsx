import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { AppState, useColorScheme } from 'react-native';
import { useLocales } from 'expo-localization';
import { useAppSnapshot } from '../data';
import { translator } from '../i18n';
import type { AppSnapshot } from '../model';

const light = {
  surface: '#FFFFFF',
  subtle: '#F5F5F7',
  selected: '#ECECFB',
  border: '#DDDDDF',
  text: '#1D1D1F',
  secondary: '#626267',
  placeholder: '#6E6E73',
  primary: '#5B5BD6',
  onPrimary: '#FFFFFF',
  error: '#B4232A',
  stale: '#9A5900',
  backlog: '#737378',
  todo: '#737378',
  doing: '#2467C3',
  blocked: '#C1353C',
  review: '#8B6600',
  done: '#248353',
  cancelled: '#737378',
};
const dark: typeof light = {
  surface: '#1C1C1E',
  subtle: '#27272B',
  selected: '#343455',
  border: '#444449',
  text: '#F5F5F7',
  secondary: '#B0B0B6',
  placeholder: '#A1A1A6',
  primary: '#AAA8FF',
  onPrimary: '#24224D',
  error: '#FF8E94',
  stale: '#F5B55B',
  backlog: '#A1A1A6',
  todo: '#A1A1A6',
  doing: '#80B3FF',
  blocked: '#FF8E94',
  review: '#F5C542',
  done: '#6DD6A0',
  cancelled: '#A1A1A6',
};
export const type = {
  headline: { fontSize: 28, lineHeight: 36, fontWeight: '700' as const },
  title: { fontSize: 20, lineHeight: 28, fontWeight: '600' as const },
  body: { fontSize: 16, lineHeight: 24 },
  label: { fontSize: 14, lineHeight: 20, fontWeight: '500' as const },
};
type UI = ReturnType<typeof translator> & {
  colors: typeof light;
  isDark: boolean;
  snapshot: AppSnapshot;
  now: number;
};
const UIContext = createContext<UI | null>(null);
export function UIProvider({ children }: { children: ReactNode }) {
  const snapshot = useAppSnapshot();
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active') setNow(Date.now());
    });
    const timer = setInterval(() => {
      if (AppState.currentState === 'active') setNow(Date.now());
    }, 60_000);
    return () => {
      clearInterval(timer);
      subscription.remove();
    };
  }, []);
  const system = useColorScheme();
  const locales = useLocales();
  const isDark =
    snapshot.settings.appearance === 'dark' ||
    (snapshot.settings.appearance === 'system' && system === 'dark');
  const language =
    snapshot.settings.language === 'system'
      ? locales[0].languageCode === 'zh'
        ? 'zh'
        : 'en'
      : snapshot.settings.language;
  const value = useMemo(
    () => ({
      ...translator(language),
      colors: isDark ? dark : light,
      isDark,
      snapshot,
      now,
    }),
    [language, isDark, snapshot, now],
  );
  return <UIContext.Provider value={value}>{children}</UIContext.Provider>;
}
export function useUI(): UI {
  const value = useContext(UIContext);
  if (!value) throw new Error('UIProvider is required');
  return value;
}
