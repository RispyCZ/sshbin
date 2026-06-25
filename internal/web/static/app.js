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

// OTP digit inputs
const otpInputs = [...document.querySelectorAll(".otp-input")];
if (otpInputs.length) {
  const otpValue = document.getElementById("otp-value");
  const otpForm = document.getElementById("otp-form");
  const otpHint = document.getElementById("otp-hint");

  function syncAndMaybeSubmit() {
    const code = otpInputs.map((i) => i.value).join("");
    otpValue.value = code;
    if (code.length === otpInputs.length) {
      otpHint.textContent = "Verifying…";
      otpForm.submit();
    }
  }

  otpInputs.forEach((input, idx) => {
    input.addEventListener("input", () => {
      const digit = input.value.replace(/\D/g, "").slice(-1);
      input.value = digit;
      if (digit && idx < otpInputs.length - 1) otpInputs[idx + 1].focus();
      syncAndMaybeSubmit();
    });

    input.addEventListener("keydown", (e) => {
      if (e.key === "Backspace" && !input.value && idx > 0) {
        otpInputs[idx - 1].value = "";
        otpInputs[idx - 1].focus();
        syncAndMaybeSubmit();
      }
    });

    input.addEventListener("paste", (e) => {
      e.preventDefault();
      const text = (e.clipboardData || window.clipboardData)
        .getData("text")
        .replace(/\D/g, "");
      text.split("").forEach((char, i) => {
        if (otpInputs[idx + i]) otpInputs[idx + i].value = char;
      });
      const last = Math.min(idx + text.length, otpInputs.length) - 1;
      otpInputs[last].focus();
      syncAndMaybeSubmit();
    });
  });

  otpInputs[0].focus();
}
