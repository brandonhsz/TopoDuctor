import React from "react";
import { TopoductorUi } from "./TopoductorUi.js";
import type { ExitPayload } from "./exit/runExit.js";

export type AppProps = {
  cwd: string;
  version: string;
  exitOutcomeRef: React.MutableRefObject<ExitPayload | null | undefined>;
};

export function App(props: AppProps) {
  return <TopoductorUi {...props} />;
}
