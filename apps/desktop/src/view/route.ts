export type Route =
  | { readonly name: "panel" }
  | { readonly name: "main"; readonly view: "board" | "settings"; readonly taskId: number | null };

const MAIN_BOARD: Route = { name: "main", view: "board", taskId: null };

/** `#/panel`, `#/main`, `#/main/task/<id>` or `#/main/settings`. Anything else opens the main board. */
export function parseRoute(hash: string): Route {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  if (parts[0] === "panel") return { name: "panel" };
  if (parts[0] !== "main") return MAIN_BOARD;
  if (parts[1] === "settings") return { name: "main", view: "settings", taskId: null };
  if (parts[1] === "task") {
    const id = Number(parts[2]);
    if (Number.isInteger(id) && id > 0) return { name: "main", view: "board", taskId: id };
  }
  return MAIN_BOARD;
}

export const taskHash = (id: number): string => `#/main/task/${id}`;
export const SETTINGS_HASH = "#/main/settings";
export const BOARD_HASH = "#/main";
