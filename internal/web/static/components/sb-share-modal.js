import { html } from '/static/lit.min.js';
import { SbModal } from '/static/components/sb-modal.js';

class SbShareModal extends SbModal {
  static properties = {
    ...SbModal.properties,
    shareId:  { type: String, attribute: 'share-id' },
    shareUrl: { type: String, attribute: 'share-url' },
    fileName: { type: String, attribute: 'file-name' },
    _copied:  { state: true },
  };

  constructor() {
    super();
    this._copied = false;
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

  renderTrigger() {
    return html`
      <button type="button" class="btn-secondary" @click=${this._show}>
        <span class="material-icons">share</span>Share
      </button>
    `;
  }

  renderBody() {
    return html`
      <div class="share-modal-body">
        <img class="share-qr-img" src="/shares/${this.shareId}/qr" alt="QR code" width="200" height="200">
        <div class="share-link">
          <input class="modal-url-input" type="text" readonly .value=${this.shareUrl ?? ''}>
          <button type="button" class="btn-copy" @click=${this._copy}>
            <span class="material-icons">${this._copied ? 'check' : 'content_copy'}</span>${this._copied ? 'Copied' : 'Copy'}
          </button>
        </div>
      </div>
    `;
  }

  render() {
    return html`
      ${this.renderTrigger()}
      ${this._visible ? html`
        <dialog class="share-modal" @close=${() => { this._visible = false; }} @click=${this._backdropClick}>
          <div class="share-modal-inner">
            <div class="share-modal-header">
              <span class="share-modal-filename">${this.fileName}</span>
              ${this.renderClose()}
            </div>
            ${this.renderBody()}
          </div>
        </dialog>
      ` : ''}
    `;
  }
}

customElements.define('sb-share-modal', SbShareModal);
