import { Pressable, Text, useWindowDimensions, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Slot, router, usePathname } from 'expo-router';
import { Icon, type IconName } from '../../ui/icons';
import { type, useUI } from '../../ui/theme';
import type { MessageKey } from '../../i18n';

const destinations: {
  route: '/' | '/agents' | '/projects' | '/calendar' | '/settings';
  label: MessageKey;
  icon: IconName;
}[] = [
  { route: '/', label: 'mine', icon: 'bulletlist' },
  { route: '/agents', label: 'agents', icon: 'robot' },
  { route: '/projects', label: 'projects', icon: 'folder' },
  { route: '/calendar', label: 'calendar', icon: 'calendar' },
  { route: '/settings', label: 'settings', icon: 'gear' },
];
export default function MainLayout() {
  const { width } = useWindowDimensions();
  const rail = width >= 720;
  const pathname = usePathname();
  const { colors, t } = useUI();
  const nav = (
    <SafeAreaView
      edges={rail ? ['top', 'bottom', 'left'] : ['bottom', 'left', 'right']}
      style={{
        backgroundColor: colors.subtle,
        ...(rail ? { width: 112 } : {}),
      }}
    >
      <View
        accessibilityRole="tablist"
        style={{
          flexDirection: rail ? 'column' : 'row',
          paddingVertical: 8,
          gap: rail ? 16 : 0,
          ...(rail ? { paddingTop: 24 } : {}),
        }}
      >
        {destinations.map((item) => {
          const selected = pathname === item.route;
          return (
            <Pressable
              key={item.route}
              testID={`${item.label}-tab`}
              accessibilityRole="tab"
              accessibilityLabel={t(item.label)}
              accessibilityState={{ selected }}
              onPress={() => {
                if (!selected) router.replace(item.route);
              }}
              style={({ pressed }) => ({
                flex: rail ? undefined : 1,
                minHeight: 64,
                alignItems: 'center',
                justifyContent: 'center',
                gap: 4,
                opacity: pressed ? 0.6 : 1,
              })}
            >
              <View
                style={{
                  backgroundColor: selected ? colors.selected : 'transparent',
                  paddingHorizontal: 16,
                  paddingVertical: 4,
                  borderRadius: 24,
                }}
              >
                <Icon
                  name={item.icon}
                  color={selected ? colors.primary : colors.secondary}
                />
              </View>
              <Text
                style={[
                  type.label,
                  {
                    fontSize: 12,
                    lineHeight: 18,
                    color: selected ? colors.primary : colors.secondary,
                    textAlign: 'center',
                  },
                ]}
              >
                {t(item.label)}
              </Text>
            </Pressable>
          );
        })}
      </View>
    </SafeAreaView>
  );
  return (
    <View
      style={{
        flex: 1,
        backgroundColor: colors.surface,
        flexDirection: rail ? 'row' : 'column',
      }}
    >
      {rail && nav}
      <View style={{ flex: 1 }}>
        <Slot />
      </View>
      {!rail && nav}
    </View>
  );
}
