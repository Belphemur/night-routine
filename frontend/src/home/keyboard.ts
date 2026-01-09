/**
 * Global keyboard event handlers
 */

import { getOptionalElement } from '../utils/dom';

/**
 * Initializes global keyboard handlers for modals
 */
export function initKeyboardHandlers(): void {
  document.addEventListener('keydown', (e) => {
    if (e.key !== 'Escape') return;

    // Try to close unlock modal
    const unlockModal = getOptionalElement('unlock-modal');
    if (unlockModal && !unlockModal.classList.contains('hidden')) {
      const cancelBtn = document.getElementById('unlock-modal-cancel');
      if (cancelBtn) {
        cancelBtn.click();
      }
      return;
    }

    // Try to close details modal
    const detailsModal = getOptionalElement('details-modal');
    if (detailsModal && !detailsModal.classList.contains('hidden')) {
      const closeBtn = document.getElementById('details-modal-close');
      if (closeBtn) {
        closeBtn.click();
      }
      return;
    }

    // Try to close sync modal (only if not loading)
    const syncModal = getOptionalElement('sync-modal');
    if (syncModal && !syncModal.classList.contains('hidden')) {
      const closeContainer = document.getElementById('sync-modal-close-container');
      if (closeContainer && !closeContainer.classList.contains('hidden')) {
        const closeBtn = document.getElementById('sync-modal-close');
        if (closeBtn) {
          closeBtn.click();
        }
      }
    }
  });
}
