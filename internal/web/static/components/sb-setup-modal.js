import { html } from '/static/lit.min.js';
import { SbModal } from '/static/components/sb-modal.js';

class SbSetupModal extends SbModal {
  static properties = {
    ...SbModal.properties,
    shareId:     { type: String,  attribute: 'share-id' },
    fileName:    { type: String,  attribute: 'file-name' },
    expires:     { type: String },
    isPublic:    { type: Boolean, attribute: 'is-public' },
    emails:      { type: String },
    hasPassword: { type: Boolean, attribute: 'has-password' },
    configured:  { type: Boolean },
    inline:      { type: Boolean },
    _private:    { state: true },
  };

  constructor() {
    super();
    this.expires     = 'never';
    this.isPublic    = false;
    this.emails      = '';
    this.hasPassword = false;
    this.configured  = false;
    this.inline      = false;
    this._private    = false;
  }

  connectedCallback() {
    super.connectedCallback();
    this._private = !this.isPublic;
  }

  _onVisibility(e) {
    this._private = e.target.value === 'private';
  }

  renderTrigger() {
    const icon = this.configured ? 'edit' : 'tune';
    const label = this.configured ? 'Edit' : 'Set up';
    return html`
      <button type="button" class="btn-secondary" @click=${this._show}>
        <span class="material-icons">${icon}</span>${label}
      </button>
    `;
  }

  renderForm() {
    const pwPlaceholder = this.hasPassword ? 'leave blank to keep current' : 'no password';
    return html`
      <form method="post" action="/setup/${this.shareId}" class="settings-form">
        ${this.inline ? '' : html`<input type="hidden" name="from" value="shares">`}
        <fieldset>
          <legend>Expiry</legend>
          <label><input type="radio" name="expires" value="1h" ?checked=${this.expires === '1h'}> 1 hour</label>
          <label><input type="radio" name="expires" value="24h" ?checked=${this.expires === '24h'}> 24 hours</label>
          <label><input type="radio" name="expires" value="168h" ?checked=${this.expires === '168h'}> 7 days</label>
          <label><input type="radio" name="expires" value="never" ?checked=${this.expires === 'never'}> Never</label>
        </fieldset>

        <fieldset>
          <legend>Visibility</legend>
          <label><input type="radio" name="visibility" value="public" ?checked=${!this._private} @change=${this._onVisibility}> Public — anyone with the link</label>
          <label><input type="radio" name="visibility" value="private" ?checked=${this._private} @change=${this._onVisibility}> Private — only people I list</label>
          <label class="field" ?hidden=${!this._private}>
            <span>Allowed emails (one per line)</span>
            <textarea name="emails" rows="3" placeholder="alice@example.com" .value=${this.emails ?? ''}></textarea>
          </label>
        </fieldset>

        <fieldset>
          <legend>Password</legend>
          <label class="field">
            <span>Extra password (optional)</span>
            <input type="password" name="password" autocomplete="new-password" placeholder="${pwPlaceholder}">
          </label>
        </fieldset>

        <div class="${this.inline ? 'form-actions' : 'sb-modal-actions'}">
          ${this.inline ? html`<a class="btn-secondary" href="/shares"><span class="material-icons">arrow_back</span>Back to my shares</a>` : html`<button type="button" class="btn-secondary" @click=${this._close}>Cancel</button>`}
          <button type="submit" class="btn-primary"><span class="material-icons">save</span>Save settings</button>
        </div>
      </form>
    `;
  }

  render() {
    if (this.inline) return this.renderForm();
    return html`
      ${this.renderTrigger()}
      ${this._visible ? html`
        <dialog class="share-modal sb-modal" @close=${() => { this._visible = false; }} @click=${this._backdropClick}>
          <div class="share-modal-inner">
            <div class="share-modal-header">
              <span class="share-modal-filename">${this.fileName}</span>
              ${this.renderClose()}
            </div>
            ${this.renderForm()}
          </div>
        </dialog>
      ` : ''}
    `;
  }
}

customElements.define('sb-setup-modal', SbSetupModal);
