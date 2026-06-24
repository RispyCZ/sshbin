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
