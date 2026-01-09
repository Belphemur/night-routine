/**
 * Sync modal management
 */

import { getRequiredElement, getOptionalElement } from '../utils/dom';
import { showModal, hideModal, type ModalElements } from '../utils/modal';
import { getLocalDateString } from '../utils/date';

interface SyncResponse {
  success: boolean;
  message?: string;
  error?: string;
}

/**
 * Initializes the sync modal and button
 */
export function initSyncModal(): void {
  const syncBtn = getOptionalElement('sync-btn');
  if (!syncBtn) return;

  const modal = getRequiredElement('sync-modal');
  const backdrop = getRequiredElement('sync-modal-backdrop');
  const panel = getRequiredElement('sync-modal-panel');
  const closeBtn = getRequiredElement('sync-modal-close');
  const closeContainer = getRequiredElement('sync-modal-close-container');

  syncBtn.addEventListener('click', () => {
    void performSync({ modal, backdrop, panel });
  });
  closeBtn.addEventListener('click', () => hideSyncModal({ modal, backdrop, panel }));

  // Close on backdrop click only if not loading
  modal.addEventListener('click', (e) => {
    if (!panel.contains(e.target as Node) && !closeContainer.classList.contains('hidden')) {
      hideSyncModal({ modal, backdrop, panel });
    }
  });
}

/**
 * Performs the calendar sync
 */
async function performSync(elements: ModalElements): Promise<void> {
  showSyncModal(elements);

  const startDate = getLocalDateString(new Date());

  try {
    const response = await fetch('/api/sync', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ start_date: startDate }),
    });

    if (!response.ok) {
      let errorMessage = `Server error: ${response.status}`;
      try {
        const errorData = await response.json() as SyncResponse;
        errorMessage = errorData.error || errorMessage;
      } catch {
        errorMessage = `${response.status} ${response.statusText}`;
      }
      showSyncError(errorMessage);
      return;
    }

    const data = await response.json() as SyncResponse;

    if (data.success) {
      showSyncSuccess(data.message || 'Your schedule has been synced successfully.');
      // Reload the page after a short delay
      setTimeout(() => {
        window.location.reload();
      }, 2000);
    } else {
      showSyncError(data.error || 'An error occurred while syncing.');
    }
  } catch (error) {
    console.error('Sync error:', error);
    showSyncError('Network error. Please check your connection and try again.');
  }
}

/**
 * Shows the sync modal in loading state
 */
function showSyncModal(elements: ModalElements): void {
  const loading = getRequiredElement('sync-loading');
  const success = getRequiredElement('sync-success');
  const error = getRequiredElement('sync-error');
  const closeContainer = getRequiredElement('sync-modal-close-container');

  // Reset to loading state
  loading.classList.remove('hidden');
  success.classList.add('hidden');
  error.classList.add('hidden');
  closeContainer.classList.add('hidden');

  showModal(elements);
}

/**
 * Shows sync success state
 */
function showSyncSuccess(message: string): void {
  const loading = getRequiredElement('sync-loading');
  const success = getRequiredElement('sync-success');
  const error = getRequiredElement('sync-error');
  const closeContainer = getRequiredElement('sync-modal-close-container');
  const messageElement = getRequiredElement('sync-success-message');

  loading.classList.add('hidden');
  success.classList.remove('hidden');
  error.classList.add('hidden');
  closeContainer.classList.remove('hidden');
  messageElement.textContent = message;
}

/**
 * Shows sync error state
 */
function showSyncError(message: string): void {
  const loading = getRequiredElement('sync-loading');
  const success = getRequiredElement('sync-success');
  const error = getRequiredElement('sync-error');
  const closeContainer = getRequiredElement('sync-modal-close-container');
  const messageElement = getRequiredElement('sync-error-message');

  loading.classList.add('hidden');
  success.classList.add('hidden');
  error.classList.remove('hidden');
  closeContainer.classList.remove('hidden');
  messageElement.textContent = message;
}

/**
 * Hides the sync modal
 */
function hideSyncModal(elements: ModalElements): void {
  hideModal(elements);
}
