import { LitElement, html, css } from '/static/lit.min.js';

const MESSAGES = {
  signed_in:  'Signed in successfully',
  signed_out: 'Signed out',
};

class SbToast extends LitElement {
  static properties = {
    _message: { state: true },
    _visible: { state: true },
  };

  static styles = css`
    :host {
      position: fixed;
      bottom: 1.5rem;
      right: 1.5rem;
      z-index: 200;
      pointer-events: none;
    }
    .toast {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: .75rem;
      background: oklch(0.18 0.030 145);
      color: oklch(0.93 0.065 145);
      border: 1px solid oklch(0.35 0.050 145);
      border-radius: var(--radius);
      padding: .75rem 1rem;
      box-shadow: 0 4px 20px oklch(0 0 0 / .3);
      min-width: 14rem;
      pointer-events: auto;
      animation: toast-in .2s ease;
    }
    @keyframes toast-in {
      from { opacity: 0; transform: translateY(6px); }
      to   { opacity: 1; transform: translateY(0); }
    }
    .close {
      background: none;
      border: 0;
      color: inherit;
      opacity: .55;
      cursor: pointer;
      padding: 0;
      font-size: 1.1rem;
      line-height: 1;
      flex-shrink: 0;
    }
    .close:hover { opacity: 1; }
  `;

  constructor() {
    super();
    this._message = '';
    this._visible = false;
    this._timer = null;
  }

  connectedCallback() {
    super.connectedCallback();
    const params = new URLSearchParams(location.search);
    const key = params.get('flash');
    if (key && MESSAGES[key]) {
      this._message = MESSAGES[key];
      this._visible = true;
      params.delete('flash');
      const qs = params.toString();
      history.replaceState(null, '', location.pathname + (qs ? '?' + qs : '') + location.hash);
      this._timer = setTimeout(() => { this._visible = false; }, 4000);
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    clearTimeout(this._timer);
  }

  _dismiss() {
    clearTimeout(this._timer);
    this._visible = false;
  }

  render() {
    if (!this._visible) return html``;
    return html`
      <div class="toast" role="status" aria-live="polite">
        <span>${this._message}</span>
        <button class="close" @click=${this._dismiss} aria-label="Dismiss">&#x2715;</button>
      </div>
    `;
  }
}

customElements.define('sb-toast', SbToast);
