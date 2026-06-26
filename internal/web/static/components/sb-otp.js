import { LitElement, html } from '/static/lit.min.js';

class SbOtp extends LitElement {
  static properties = {
    digits:       { type: Number },
    formId:       { type: String, attribute: 'form-id' },
    valueInputId: { type: String, attribute: 'value-input-id' },
  };

  constructor() {
    super();
    this.digits = 6;
  }

  createRenderRoot() {
    return this;
  }

  render() {
    return html`
      <div class="otp-container">
        ${Array.from({ length: this.digits }, (_, i) => html`
          <input
            class="otp-input"
            type="text"
            inputmode="numeric"
            maxlength="1"
            autocomplete="${i === 0 ? 'one-time-code' : 'off'}"
            aria-label="Digit ${i + 1}">
        `)}
      </div>
      <p class="otp-hint">Enter code to continue</p>
    `;
  }

  firstUpdated() {
    const inputs = [...this.querySelectorAll('.otp-input')];
    const valueInput = document.getElementById(this.valueInputId);
    const form = document.getElementById(this.formId);
    const hint = this.querySelector('.otp-hint');

    const syncAndSubmit = () => {
      const code = inputs.map(i => i.value).join('');
      valueInput.value = code;
      if (code.length === inputs.length) {
        hint.textContent = 'Verifying…';
        form.submit();
      }
    };

    inputs.forEach((input, idx) => {
      input.addEventListener('input', () => {
        input.value = input.value.replace(/\D/g, '').slice(-1);
        if (input.value && idx < inputs.length - 1) inputs[idx + 1].focus();
        syncAndSubmit();
      });

      input.addEventListener('keydown', e => {
        if (e.key === 'Backspace' && !input.value && idx > 0) {
          inputs[idx - 1].value = '';
          inputs[idx - 1].focus();
          syncAndSubmit();
        }
      });

      input.addEventListener('paste', e => {
        e.preventDefault();
        const text = (e.clipboardData || window.clipboardData)
          .getData('text')
          .replace(/\D/g, '');
        text.split('').forEach((char, i) => {
          if (inputs[idx + i]) inputs[idx + i].value = char;
        });
        inputs[Math.min(idx + text.length, inputs.length) - 1].focus();
        syncAndSubmit();
      });
    });

    inputs[0]?.focus();
  }
}

customElements.define('sb-otp', SbOtp);
