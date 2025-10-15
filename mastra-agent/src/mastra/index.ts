import { Mastra } from "@mastra/core/mastra";
import { PinoLogger } from "@mastra/loggers";
import { LibSQLStore } from "@mastra/libsql";
// import { weatherWorkflow } from "./workflows/songWorkflow";
import { songAgent } from "./agents/songAgent";

export const mastra = new Mastra({
  // workflows: { weatherWorkflow },
  agents: { songAgent },
  

  storage: new LibSQLStore({
    // stores observability, scores, ... into memory storage
    // for persistence, use file:../mastra.db
    url: ":memory:",
  }),

  logger: new PinoLogger({
    name: "Mastra",
    level: "info",
  }),

  telemetry: {
    // deprecated as of Nov 4 release
    enabled: false,
  },

  observability: {
    // Enables tracing in Playground
    default: { enabled: true },
  },
});