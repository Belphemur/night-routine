/**
 * Modal animation utilities
 */

export interface ModalElements {
  modal: HTMLElement;
  backdrop: HTMLElement;
  panel: HTMLElement;
}

/**
 * Shows a modal with smooth animations
 */
export function showModal(elements: ModalElements, focusElement?: HTMLElement): void {
  const { modal, backdrop, panel } = elements;
  
  modal.classList.remove('hidden');
  
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      backdrop.classList.remove('opacity-0');
      backdrop.classList.add('opacity-100');
      
      panel.classList.remove('opacity-0', 'translate-y-4', 'sm:translate-y-0', 'sm:scale-95');
      panel.classList.add('opacity-100', 'translate-y-0', 'sm:scale-100');
    });
  });
  
  if (focusElement) {
    setTimeout(() => focusElement.focus(), 100);
  }
}

/**
 * Hides a modal with smooth animations
 */
export function hideModal(elements: ModalElements, onComplete?: () => void): void {
  const { modal, backdrop, panel } = elements;
  
  backdrop.classList.remove('opacity-100');
  backdrop.classList.add('opacity-0');
  
  panel.classList.remove('opacity-100', 'translate-y-0', 'sm:scale-100');
  panel.classList.add('opacity-0', 'translate-y-4', 'sm:translate-y-0', 'sm:scale-95');
  
  panel.addEventListener('transitionend', function handler() {
    modal.classList.add('hidden');
    panel.removeEventListener('transitionend', handler);
    if (onComplete) {
      onComplete();
    }
  }, { once: true });
}

/**
 * Sets up backdrop click handler to close modal
 */
export function setupBackdropClose(
  modal: HTMLElement,
  panel: HTMLElement,
  onClose: () => void
): void {
  modal.addEventListener('click', (e) => {
    if (!panel.contains(e.target as Node)) {
      onClose();
    }
  });
}
