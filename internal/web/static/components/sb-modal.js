import { LitElement, html } from '/static/lit.min.js';

export class SbModal extends LitElement {
  static properties = {
    heading:      { type: String },
    message:      { type: String },
    formAction:   { type: String, attribute: 'form-action' },
    confirmLabel: { type: String, attribute: 'confirm-label' },
    triggerLabel: { type: String, attribute: 'trigger-label' },
    triggerIcon:  { type: String, attribute: 'trigger-icon' },
    danger:       { type: Boolean },
    _visible:     { state: true },
  };

  createRenderRoot() { return this; }

  constructor() {
    super();
    this.heading      = 'Confirm';
    this.confirmLabel = 'Confirm';
    this.triggerLabel = 'Confirm';
    this.danger       = false;
    this._visible     = false;
  }

  _show() {
    this._visible = true;
    this.updateComplete.then(() => this.querySelector('dialog')?.showModal());
  }

  _close() {
    this.querySelector('dialog')?.close();
    this._visible = false;
  }

  _backdropClick(e) {
    if (e.target.tagName === 'DIALOG') this._close();
  }

  renderClose() {
    return html`
      <button type="button" class="modal-close-btn" aria-label="Close" @click=${this._close}>
        <svg class="modal-close-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"
          stroke-width="2.5" stroke-linecap="round" aria-hidden="true">
          <line x1="6" y1="6" x2="18" y2="18" />
          <line x1="18" y1="6" x2="6" y2="18" />
        </svg>
      </button>
    `;
  }

  renderTrigger() {
    const cls = this.danger ? 'btn-danger' : 'btn-secondary';
    return html`
      <button type="button" class="${cls}" @click=${this._show}>
        ${this.triggerIcon ? html`<span class="material-icons">${this.triggerIcon}</span>` : ''}${this.triggerLabel}
      </button>
    `;
  }

  renderBody() {
    const cls = this.danger ? 'btn-danger' : 'btn-primary';
    return html`
      <div class="sb-modal-body">
        <p class="sb-modal-message">${this.message}</p>
        <div class="sb-modal-actions">
          <button type="button" class="btn-secondary" @click=${this._close}>Cancel</button>
          <button type="submit" class="${cls}">${this.confirmLabel}</button>
        </div>
      </div>
    `;
  }

  render() {
    return html`
      ${this.renderTrigger()}
      ${this._visible ? html`
        <dialog class="share-modal sb-modal" @close=${() => { this._visible = false; }} @click=${this._backdropClick}>
          <form method="post" action="${this.formAction}">
            <div class="share-modal-inner">
              <div class="share-modal-header">
                <span class="share-modal-filename">${this.heading}</span>
                ${this.renderClose()}
              </div>
              ${this.renderBody()}
            </div>
          </form>
        </dialog>
      ` : ''}
    `;
  }
}

customElements.define('sb-modal', SbModal);
