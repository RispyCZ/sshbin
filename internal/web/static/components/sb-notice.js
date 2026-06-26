import { LitElement, html, css } from '/static/lit.min.js';

class SbNotice extends LitElement {
  static properties = {
    type: { type: String },
  };

  static styles = css`
    :host { display: block; }
    .notice {
      border-radius: var(--radius);
      padding: 1rem 1.25rem;
      margin-bottom: 1rem;
    }
    .notice.error {
      background: color-mix(in srgb, var(--danger) 14%, transparent);
      border: 1px solid var(--danger);
    }
    .notice.success {
      background: color-mix(in srgb, var(--accent) 14%, transparent);
      border: 1px solid var(--accent);
    }
  `;

  render() {
    return html`<div class="notice ${this.type}"><slot></slot></div>`;
  }
}

customElements.define('sb-notice', SbNotice);
