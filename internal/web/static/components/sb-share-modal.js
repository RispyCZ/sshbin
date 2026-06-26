import { LitElement, html } from '/static/lit.min.js';

class SbShareModal extends LitElement {
  static properties = {
    shareId:  { type: String, attribute: 'share-id' },
    shareUrl: { type: String, attribute: 'share-url' },
    fileName: { type: String, attribute: 'file-name' },
    _visible: { state: true },
    _copied:  { state: true },
  };

  createRenderRoot() { return this; }

  constructor() {
    super();
    this._visible = false;
    this._copied  = false;
  }

  _show() {
    this._visible = true;
    this.updateComplete.then(() => this.querySelector('dialog')?.showModal());
  }

  _close() {
    this.querySelector('dialog')?.close();
    this._visible = false;
  }

  async _copy() {
    try {
      await navigator.clipboard.writeText(this.shareUrl);
      this._copied = true;
      setTimeout(() => { this._copied = false; }, 1500);
    } catch {
      this.querySelector('.modal-url-input')?.select();
    }
  }

  _backdropClick(e) {
    if (e.target.tagName === 'DIALOG') this._close();
  }

  render() {
    return html`
      <button type="button" class="btn-secondary" @click=${this._show}>Share</button>
      ${this._visible ? html`
        <dialog class="share-modal" @close=${() => { this._visible = false; }} @click=${this._backdropClick}>
          <div class="share-modal-inner">
            <div class="share-modal-header">
              <span class="share-modal-filename">${this.fileName}</span>
              <button type="button" class="modal-close-btn" aria-label="Close" @click=${this._close}>
                <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
                  <line x1="1" y1="1" x2="11" y2="11"/><line x1="11" y1="1" x2="1" y2="11"/>
                </svg>
              </button>
            </div>
            <div class="share-modal-body">
              <img class="share-qr-img" src="/shares/${this.shareId}/qr" alt="QR code" width="200" height="200">
              <div class="share-link">
                <input class="modal-url-input" type="text" readonly .value=${this.shareUrl ?? ''}>
                <button type="button" class="btn-copy" @click=${this._copy}>
                  ${this._copied ? 'Copied' : 'Copy'}
                </button>
              </div>
            </div>
          </div>
        </dialog>
      ` : ''}
    `;
  }
}

customElements.define('sb-share-modal', SbShareModal);
