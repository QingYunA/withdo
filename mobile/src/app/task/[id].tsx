import { useLocalSearchParams } from 'expo-router';
import { TaskEditor } from '../../screens/TaskEditor';
export default function TaskScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <TaskEditor key={id} taskId={id} />;
}
