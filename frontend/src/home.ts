/**
 * Home page main script
 * Initializes all home page functionality
 */

import { initDesktopCalendar } from './home/desktopCalendar';
import { initMobileCalendar } from './home/mobileCalendar';
import { initUnlockModal } from './home/unlockModal';
import { initDetailsModal } from './home/detailsModal';
import { initSyncModal } from './home/syncModal';
import { initKeyboardHandlers } from './home/keyboard';

/**
 * Main initialization function
 */
function initHome(): void {
  // Initialize desktop calendar
  initDesktopCalendar();

  // Initialize mobile calendar
  initMobileCalendar();

  // Initialize modals
  initUnlockModal();
  initDetailsModal();
  initSyncModal();

  // Initialize global keyboard handlers
  initKeyboardHandlers();
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', initHome);
