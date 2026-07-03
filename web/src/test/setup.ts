import "@testing-library/jest-dom/vitest";
import { afterEach } from "vite-plus/test";
import { cleanup } from "@testing-library/react";

// Unmount React trees between tests so the jsdom document stays isolated.
afterEach(() => {
  cleanup();
});
