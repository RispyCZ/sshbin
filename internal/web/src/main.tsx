import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { App } from "./components/App.tsx";
import { ColorModeProvider } from "./components/ColorModeProvider.tsx";
import { NotifyProvider } from "./components/NotifyProvider.tsx";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ColorModeProvider>
      <NotifyProvider>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </NotifyProvider>
    </ColorModeProvider>
  </StrictMode>,
);
