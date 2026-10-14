export type Route = { readonly name: "panel" } | { readonly name: "main"; readonly taskId: number | null };

const MAIN_BOARD: Route = { name: "main", taskId: null };

/** `#/panel`, `#/main` or `#/main/task/<id>` (opens that task). Anything else opens the main board. */
export function parseRoute(hash: string): Route {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  if (parts[0] === "panel") return { name: "panel" };
  if (parts[0] !== "main") return MAIN_BOARD;
  if (parts[1] === "task") {
    const id = Number(parts[2]);
    if (Number.isInteger(id) && id > 0) return { name: "main", taskId: id };
  }
  return MAIN_BOARD;
}

export const taskHash = (id: number): string => `#/main/task/${id}`;
export const BOARD_HASH = "#/main";
