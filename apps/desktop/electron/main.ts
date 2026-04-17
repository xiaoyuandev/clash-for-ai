import { startGoProcess } from "./go-process";
import { setupTray } from "./tray";

function main() {
  const goHandle = startGoProcess();
  setupTray();

  process.on("exit", () => {
    goHandle.stop();
  });
}

main();
