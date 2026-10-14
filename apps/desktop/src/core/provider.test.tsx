import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, test, vi } from "vitest";
import { CoreProvider } from "./provider";
import { useCore } from "./context";

function Probe() {
  const { endpoint } = useCore();
  return <p>connected to {endpoint.baseUrl}</p>;
}

const endpoint = { baseUrl: "http://127.0.0.1:9", token: "t" };

describe("CoreProvider", () => {
  test("shows a loading state, then renders children once the endpoint is known", async () => {
    render(
      <CoreProvider discover={async () => endpoint} watchErrors={() => () => {}}>
        <Probe />
      </CoreProvider>,
    );
    expect(screen.getByText(/starting/i)).toBeInTheDocument();
    expect(await screen.findByText("connected to http://127.0.0.1:9")).toBeInTheDocument();
  });

  test("shows the failure and retries on request", async () => {
    const discover = vi.fn().mockRejectedValueOnce(new Error("adhd-core exited")).mockResolvedValue(endpoint);
    render(
      <CoreProvider discover={discover} watchErrors={() => () => {}}>
        <Probe />
      </CoreProvider>,
    );
    expect(await screen.findByText(/adhd-core exited/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /try again/i }));
    expect(await screen.findByText(/connected to/)).toBeInTheDocument();
    expect(discover).toHaveBeenCalledTimes(2);
  });

  test("switches to the error state when the shell reports a sidecar failure", async () => {
    let report: (message: string) => void = () => {};
    render(
      <CoreProvider discover={async () => endpoint} watchErrors={(cb) => ((report = cb), () => {})}>
        <Probe />
      </CoreProvider>,
    );
    await screen.findByText(/connected to/);
    act(() => report("adhd-core crashed twice"));
    await waitFor(() => expect(screen.getByText(/adhd-core crashed twice/)).toBeInTheDocument());
  });
});
