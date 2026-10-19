import { useState, type ReactNode } from 'react';
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
  type TextInputProps,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { Icon, type IconName } from './icons';
import { type, useUI } from './theme';

export function Label({
  children,
  secondary = false,
  style,
}: {
  children: ReactNode;
  secondary?: boolean;
  style?: StyleProp<import('react-native').TextStyle>;
}) {
  const { colors } = useUI();
  return (
    <Text
      style={[
        type.body,
        { color: secondary ? colors.secondary : colors.text },
        style,
      ]}
    >
      {children}
    </Text>
  );
}
export function Button({
  label,
  onPress,
  kind = 'primary',
  disabled = false,
  busy = false,
  icon,
  testID,
}: {
  label: string;
  onPress: () => void;
  kind?: 'primary' | 'text' | 'danger' | 'tonal';
  disabled?: boolean;
  busy?: boolean;
  icon?: IconName;
  testID?: string;
}) {
  const { colors } = useUI();
  const foreground =
    kind === 'primary'
      ? colors.onPrimary
      : kind === 'danger'
        ? colors.error
        : colors.primary;
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: disabled || busy, busy }}
      disabled={disabled || busy}
      onPress={onPress}
      style={({ pressed }) => [
        styles.button,
        {
          backgroundColor:
            kind === 'primary'
              ? colors.primary
              : kind === 'tonal'
                ? colors.selected
                : 'transparent',
          opacity: disabled ? 0.45 : pressed ? 0.7 : 1,
        },
      ]}
    >
      {busy ? (
        <ActivityIndicator color={foreground} />
      ) : icon ? (
        <Icon name={icon} color={foreground} />
      ) : null}
      <Text
        style={[
          type.label,
          { color: foreground, textAlign: 'center', flexShrink: 1 },
        ]}
      >
        {label}
      </Text>
    </Pressable>
  );
}
export function IconButton({
  name,
  label,
  onPress,
  disabled = false,
  testID,
}: {
  name: IconName;
  label: string;
  onPress: () => void;
  disabled?: boolean;
  testID?: string;
}) {
  const { colors } = useUI();
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      disabled={disabled}
      accessibilityState={{ disabled }}
      onPress={onPress}
      style={({ pressed }) => [
        styles.iconButton,
        {
          backgroundColor: pressed ? colors.selected : 'transparent',
          opacity: disabled ? 0.4 : 1,
        },
      ]}
    >
      <Icon name={name} />
    </Pressable>
  );
}
export function Page({
  children,
  scroll = true,
  title,
  back = false,
  right,
  bottom,
  padding = true,
}: {
  children: ReactNode;
  scroll?: boolean;
  title: string;
  back?: boolean;
  right?: ReactNode;
  bottom?: ReactNode;
  padding?: boolean;
}) {
  const { colors, t } = useUI();
  const body = (
    <View style={[styles.body, { maxWidth: 900 }, padding && styles.padding]}>
      {children}
    </View>
  );
  return (
    <SafeAreaView
      edges={['top', 'left', 'right']}
      style={{ flex: 1, backgroundColor: colors.surface }}
    >
      <View style={[styles.toolbar, { borderBottomColor: colors.border }]}>
        {back && (
          <IconButton
            testID="back-button"
            name="chevron-left"
            label={t('back')}
            onPress={() =>
              router.canGoBack() ? router.back() : router.replace('/')
            }
          />
        )}
        <Text
          accessibilityRole="header"
          style={[
            back ? type.title : type.headline,
            { color: colors.text, flex: 1, marginLeft: back ? 0 : 8 },
          ]}
        >
          {title}
        </Text>
        {right}
      </View>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        {scroll ? (
          <ScrollView
            keyboardShouldPersistTaps="handled"
            contentContainerStyle={{
              flexGrow: 1,
              paddingBottom: bottom ? 8 : 32,
            }}
          >
            {body}
          </ScrollView>
        ) : (
          children
        )}
        {bottom && (
          <SafeAreaView
            edges={['bottom']}
            style={[
              styles.footer,
              {
                backgroundColor: colors.surface,
                borderTopColor: colors.border,
              },
            ]}
          >
            {bottom}
          </SafeAreaView>
        )}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
export function Field({
  label,
  multiline,
  style,
  ...props
}: TextInputProps & { label?: string }) {
  const { colors } = useUI();
  return (
    <View style={{ gap: 8 }}>
      {label && (
        <Text style={[type.label, { color: colors.secondary }]}>{label}</Text>
      )}
      <TextInput
        accessibilityLabel={label ?? props.placeholder}
        placeholderTextColor={colors.placeholder}
        selectionColor={colors.primary}
        multiline={multiline}
        {...props}
        style={[
          type.body,
          styles.input,
          {
            color: colors.text,
            backgroundColor: colors.subtle,
            minHeight: multiline ? 112 : 56,
            textAlignVertical: multiline ? 'top' : 'center',
          },
          style,
        ]}
      />
    </View>
  );
}
export function Section({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  const { colors } = useUI();
  return (
    <View style={{ gap: 12, marginTop: 20 }}>
      <Text
        accessibilityRole="header"
        style={[type.label, { color: colors.secondary }]}
      >
        {title}
      </Text>
      {children}
    </View>
  );
}
export function Choice<T extends string>({
  options,
  selected,
  onSelect,
  label,
}: {
  options: { value: T; label: string }[];
  selected: T;
  onSelect: (value: T) => void;
  label: string;
}) {
  const { colors } = useUI();
  return (
    <View
      accessibilityLabel={label}
      style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}
    >
      {options.map((option) => (
        <Pressable
          accessibilityRole="radio"
          accessibilityState={{ selected: selected === option.value }}
          accessibilityLabel={option.label}
          key={option.value}
          onPress={() => onSelect(option.value)}
          style={({ pressed }) => [
            styles.chip,
            {
              borderColor:
                selected === option.value ? colors.primary : colors.border,
              backgroundColor:
                selected === option.value
                  ? colors.selected
                  : pressed
                    ? colors.subtle
                    : 'transparent',
            },
          ]}
        >
          <Text
            style={[
              type.label,
              {
                color:
                  selected === option.value ? colors.primary : colors.secondary,
              },
            ]}
          >
            {option.label}
          </Text>
        </Pressable>
      ))}
    </View>
  );
}
export function Empty({
  title,
  body,
  icon = 'inbox',
  action,
}: {
  title: string;
  body?: string;
  icon?: IconName;
  action?: ReactNode;
}) {
  const { colors } = useUI();
  return (
    <View style={styles.empty}>
      <Icon name={icon} size={48} color={colors.primary} />
      <Text
        accessibilityRole="header"
        style={[type.title, { color: colors.text, textAlign: 'center' }]}
      >
        {title}
      </Text>
      {body && (
        <Label secondary style={{ textAlign: 'center', maxWidth: 420 }}>
          {body}
        </Label>
      )}
      {action}
    </View>
  );
}
export function ErrorMessage({
  message,
}: {
  message: string | null | undefined;
}) {
  const { colors } = useUI();
  if (!message) return null;
  return (
    <Text
      accessibilityRole="alert"
      accessibilityLiveRegion="polite"
      selectable
      style={[type.body, { color: colors.error, marginVertical: 12 }]}
    >
      {message}
    </Text>
  );
}
export function useAction() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const run = async (action: () => Promise<void>) => {
    if (busy) return;
    setBusy(true);
    setError(null);
    try {
      await action();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };
  return { busy, error, run, setError };
}
export function Row({
  children,
  style,
}: {
  children: ReactNode;
  style?: StyleProp<ViewStyle>;
}) {
  return (
    <View
      style={[{ flexDirection: 'row', alignItems: 'center', gap: 8 }, style]}
    >
      {children}
    </View>
  );
}
export const styles = StyleSheet.create({
  body: { width: '100%', alignSelf: 'center', flexGrow: 1 },
  padding: { paddingHorizontal: 20, paddingTop: 12, gap: 16 },
  toolbar: {
    minHeight: 72,
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 12,
    gap: 8,
  },
  button: {
    minHeight: 48,
    paddingHorizontal: 20,
    paddingVertical: 12,
    borderRadius: 24,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
  },
  iconButton: {
    minWidth: 48,
    minHeight: 48,
    justifyContent: 'center',
    alignItems: 'center',
    borderRadius: 24,
  },
  input: { borderRadius: 12, paddingHorizontal: 16, paddingVertical: 14 },
  chip: {
    minHeight: 48,
    justifyContent: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderWidth: 1,
    borderRadius: 24,
  },
  empty: {
    flexGrow: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 24,
    paddingVertical: 64,
    gap: 16,
  },
  footer: {
    borderTopWidth: StyleSheet.hairlineWidth,
    paddingHorizontal: 20,
    paddingTop: 12,
    paddingBottom: 12,
    gap: 8,
  },
});
