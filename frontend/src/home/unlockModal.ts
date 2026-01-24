/**
 * Unlock modal management
 */

import { getRequiredElement } from '../utils/dom';
import { showModal, hideModal, setupBackdropClose, type ModalElements } from '../utils/modal';

let currentAssignmentId: string | null = null;

/**
 * Initializes the unlock modal
 */
export function initUnlockModal(): void {
  const modal = getRequiredElement('unlock-modal');
  const backdrop = getRequiredElement('unlock-modal-backdrop');
  const panel = getRequiredElement('unlock-modal-panel');
  const cancelBtn = getRequiredElement('unlock-modal-cancel');
  const confirmBtn = getRequiredElement('unlock-modal-confirm');

  const elements: ModalElements = { modal, backdrop, panel };

  cancelBtn.addEventListener('click', () => hideUnlockModal(elements));
  confirmBtn.addEventListener('click', () => {
    unlockAssignment();
    hideUnlockModal(elements);
  });

  setupBackdropClose(modal, panel, () => hideUnlockModal(elements));
}

/**
 * Shows the unlock modal for a specific assignment
 */
export function showUnlockModal(assignmentId: string): void {
  currentAssignmentId = assignmentId;
  
  const modal = getRequiredElement('unlock-modal');
  const backdrop = getRequiredElement('unlock-modal-backdrop');
  const panel = getRequiredElement('unlock-modal-panel');
  const confirmBtn = getRequiredElement('unlock-modal-confirm');

  showModal({ modal, backdrop, panel }, confirmBtn);
}

/**
 * Hides the unlock modal
 */
function hideUnlockModal(elements: ModalElements): void {
  hideModal(elements, () => {
    currentAssignmentId = null;
  });
}

/**
 * Submits the unlock form
 */
function unlockAssignment(): void {
  if (!currentAssignmentId) return;

  const form = document.createElement('form');
  form.method = 'POST';
  form.action = '/unlock';

  const input = document.createElement('input');
  input.type = 'hidden';
  input.name = 'assignment_id';
  input.value = currentAssignmentId;

  form.appendChild(input);
  document.body.appendChild(form);
  form.submit();
}
