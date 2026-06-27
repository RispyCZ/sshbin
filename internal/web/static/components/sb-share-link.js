import { LitElement, html } from '/static/lit.min.js';

class SbShareLink extends LitElement {
  static properties = {
    value:   { type: String },
    _copied: { state: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this._copied = false;
  }

  async _handleCopy() {
    try {
      await navigator.clipboard.writeText(this.value);
      this._copied = true;
      setTimeout(() => { this._copied = false; }, 1500);
    } catch {
      this.querySelector('input')?.select();
    }
  }

  render() {
    return html`
      <div class="share-link">
        <input type="text" readonly .value=${this.value ?? ''}>
        <button type="button" class="btn-copy" @click=${this._handleCopy}>
          <span class="material-icons">${this._copied ? 'check' : 'content_copy'}</span>${this._copied ? 'Copied' : 'Copy'}
        </button>
      </div>
    `;
  }
}

customElements.define('sb-share-link', SbShareLink);
