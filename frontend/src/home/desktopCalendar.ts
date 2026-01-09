/**
 * Desktop calendar initialization
 */

import { getLocalDateString } from '../utils/date';
import { getOptionalElement } from '../utils/dom';
import { showUnlockModal } from './unlockModal';
import { showDetailsModal } from './detailsModal';

/**
 * Initializes the desktop calendar with today highlighting and click handlers
 */
export function initDesktopCalendar(): void {
  const calendar = getOptionalElement('assignment-calendar');
  if (!calendar) return;

  // Highlight today's cell
  const today = new Date();
  const todayString = getLocalDateString(today);
  const todayCell = calendar.querySelector<HTMLTableCellElement>(`td[data-date="${todayString}"]`);

  if (todayCell) {
    todayCell.classList.add('today-cell');
  }

  // Handle clicks on calendar cells
  calendar.addEventListener('click', (e) => {
    const cell = (e.target as HTMLElement).closest<HTMLTableCellElement>('td[data-assignment-id]');
    if (!cell) return;

    e.stopPropagation();
    const assignmentId = cell.dataset.assignmentId;
    if (!assignmentId) return;

    // Check if this is an overridden cell
    if (cell.classList.contains('overridden')) {
      showUnlockModal(assignmentId);
    } else {
      showDetailsModal(assignmentId);
    }
  });
}
