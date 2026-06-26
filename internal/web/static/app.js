// Theme toggle
const themeToggle = document.getElementById("theme-toggle");
if (themeToggle) {
  const root = document.documentElement;
  const systemDark = () => window.matchMedia("(prefers-color-scheme: dark)").matches;
  function currentTheme() {
    return root.getAttribute("data-theme") || (systemDark() ? "dark" : "light");
  }
  themeToggle.addEventListener("click", () => {
    const next = currentTheme() === "dark" ? "light" : "dark";
    root.setAttribute("data-theme", next);
    localStorage.setItem("theme", next);
  });
}

// Toggle the allowed-emails field based on the selected visibility.
const visibilityInputs = document.querySelectorAll('input[name="visibility"]');
const privateOnly = document.querySelector("[data-private-only]");
function syncVisibility() {
  if (!privateOnly) return;
  const selected = document.querySelector('input[name="visibility"]:checked');
  privateOnly.hidden = !selected || selected.value !== "private";
}
visibilityInputs.forEach((i) => i.addEventListener("change", syncVisibility));
syncVisibility();
