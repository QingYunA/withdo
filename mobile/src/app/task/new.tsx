import { useLocalSearchParams } from 'expo-router';
import { TaskEditor } from '../../screens/TaskEditor';
export default function NewTaskScreen() {
  const { projectId, parentId, dueAt } = useLocalSearchParams<{
    projectId?: string;
    parentId?: string;
    dueAt?: string;
  }>();
  return <TaskEditor projectId={projectId} parentId={parentId} dueAt={dueAt} />;
}
