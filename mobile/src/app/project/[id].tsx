import { useLocalSearchParams } from 'expo-router';
import { ProjectScreen } from '../../screens/Projects';
export default function ProjectDetail() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <ProjectScreen key={id} id={id} />;
}
