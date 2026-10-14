import { PanelView } from "./panel/PanelView";
import { useHashRoute } from "./ui/use-hash-route";

export function App() {
  const route = useHashRoute();
  if (route.name === "panel") return <PanelView />;
  return <p className="p-4 text-muted">Main window</p>;
}
