// User menu dropdown
const userToggle = document.querySelector(".user-toggle");
const userDropdown = document.querySelector(".user-dropdown");
if (userToggle && userDropdown) {
  userToggle.addEventListener("click", (e) => {
    e.stopPropagation();
    const open = userToggle.getAttribute("aria-expanded") === "true";
    userToggle.setAttribute("aria-expanded", String(!open));
    userDropdown.hidden = open;
  });
  document.addEventListener("click", () => {
    userToggle.setAttribute("aria-expanded", "false");
    userDropdown.hidden = true;
  });
  userDropdown.addEventListener("click", (e) => e.stopPropagation());
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

document.querySelectorAll("[data-copy-btn]").forEach((btn) => {
  btn.addEventListener("click", async () => {
    const input = btn.parentElement.querySelector("[data-copy]");
    if (!input) return;
    try {
      await navigator.clipboard.writeText(input.value);
      const original = btn.textContent;
      btn.textContent = "Copied";
      setTimeout(() => (btn.textContent = original), 1500);
    } catch {
      input.select();
    }
  });
});
