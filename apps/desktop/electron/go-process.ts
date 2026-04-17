import { spawn, type ChildProcess } from "node:child_process";
import { join } from "node:path";

export interface GoProcessHandle {
  process: ChildProcess;
  stop: () => void;
}

export function startGoProcess() {
  const entry = join(process.cwd(), "core", "cmd", "clash-for-ai-core", "main.go");
  const child = spawn("go", ["run", entry], {
    stdio: "inherit"
  });

  return {
    process: child,
    stop() {
      child.kill();
    }
  } satisfies GoProcessHandle;
}
