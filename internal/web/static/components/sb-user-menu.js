import { LitElement, html } from '/static/lit.min.js';

class SbUserMenu extends LitElement {
  static properties = {
    email: { type: String },
    _open: { state: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this._open = false;
    this._closeOnOutsideClick = () => { this._open = false; };
  }

  connectedCallback() {
    super.connectedCallback();
    document.addEventListener('click', this._closeOnOutsideClick);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener('click', this._closeOnOutsideClick);
  }

  _toggle(e) {
    e.stopPropagation();
    this._open = !this._open;
  }

  render() {
    return html`
      <div class="user-menu">
        <button class="user-toggle" type="button"
          aria-expanded=${String(this._open)}
          aria-haspopup="true"
          @click=${this._toggle}>
          <span class="user-avatar">${this.email?.[0] ?? ''}</span>
          <span class="user-email">${this.email}</span>
          <svg class="chevron" width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
            <path d="M2 3.5l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
        <div class="user-dropdown" ?hidden=${!this._open} @click=${e => e.stopPropagation()}>
          <span class="dropdown-email">${this.email}</span>
          <a class="dropdown-item" href="/shares">My shares</a>
          <form method="post" action="/logout">
            <button type="submit" class="dropdown-item">Sign out</button>
          </form>
        </div>
      </div>
    `;
  }
}

customElements.define('sb-user-menu', SbUserMenu);
