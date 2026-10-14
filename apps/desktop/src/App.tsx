import { MainView } from "./main/MainView";
import { PanelView } from "./panel/PanelView";
import { useHashRoute } from "./ui/use-hash-route";

export function App() {
  const route = useHashRoute();
  return route.name === "panel" ? <PanelView /> : <MainView route={route} />;
}
